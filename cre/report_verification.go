package cre

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/known/anypb"
)

var keyCache = &sync.Map{}

type donInfo struct {
	f int
	// signers maps signer address → nodeOperatorId (slot 0 of NodeInfo ABI tuple).
	signers map[common.Address]uint32
}

type Environment struct {
	ChainSelector   uint64
	RegistryAddress string
}

// Zone is a specific CRE instance that has its own workflow DON.
// An environment can contain many zones. Most workflow authors should verify reports against
// ProductionEnvironment using ParseReport and Report.Verify, a Zones may be specified to lock down which production
// environments are allowed to sign.
type Zone struct {
	Environment Environment
	DonID       uint32
}

func ProductionEnvironment() Environment {
	return Environment{
		ChainSelector:   5009297550715157269,
		RegistryAddress: "0x76c9cf548b4179F8901cda1f8623568b58215E62",
	}
}

func ZoneFromEnvironment(env Environment, donId uint32) Zone {
	return Zone{Environment: env, DonID: donId}
}

type ReportParseConfig struct {
	AcceptedZones        []Zone
	AcceptedEnvironments []Environment
	// SkipSignatureVerification skips the signature verification step. This can be used for testing or in environments where trust is established by other means, but should be used with caution as it disables a critical security check.
	// It should only be used alongside Report.Verify if filtering reports first to avoid unnecessary calls to the blockchain.
	SkipSignatureVerification bool
}

var defaultVerificationConfig = ReportParseConfig{AcceptedEnvironments: []Environment{ProductionEnvironment()}}

// ParseReport parses a CRE report and verifies it against the production CRE environment.
// The first time a DON's report is seen, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReport(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte) (*Report, error) {
	return ParseReportWithConfig(runtime, rawReport, signatures, reportContext, defaultVerificationConfig)
}

// ParseReportWithConfig parses a CRE report and verifies it as specified by the config.
// The first time a DON's report is seen, and SkipSignatureVerification is false, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReportWithConfig(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte, config ReportParseConfig) (*Report, error) {
	attrSigs := make([]*sdk.AttributedSignature, len(signatures))
	for i, s := range signatures {
		attrSigs[i] = &sdk.AttributedSignature{Signature: s}
	}

	// Extract ConfigDigest and SeqNr from the report context when present.
	// Standard layout: bytes 0-31 = ConfigDigest, bytes 32-39 = SeqNr (big-endian uint64).
	var configDigest []byte
	var seqNr uint64
	if len(reportContext) >= 40 {
		configDigest = reportContext[:32]
		seqNr = binary.BigEndian.Uint64(reportContext[32:40])
	}

	report := &Report{
		report: &sdk.ReportResponse{
			RawReport:     rawReport,
			Sigs:          attrSigs,
			ReportContext: reportContext,
			ConfigDigest:  configDigest,
			SeqNr:         seqNr,
		},
	}

	if config.SkipSignatureVerification {
		if _, err := report.parseHeader(); err != nil {
			return nil, err
		}

		return report, nil
	}

	if err := report.VerifySignaturesWithConfig(runtime, config); err != nil {
		return nil, err
	}

	return report, nil
}

// VerifySignatures verifies the signatures on a Report against the CRE's production environment.
// VerifySignatures only needs to be called if SkipSignatureVerification was used with ParseReportWithConfig.
// The first time a DON's report is seen, the signatures will be fetched from the chain. It will be cached for later report parsing.
func (r *Report) VerifySignatures(runtime Runtime) error {
	return r.VerifySignaturesWithConfig(runtime, defaultVerificationConfig)
}

// VerifySignaturesWithConfig verifies the signatures on a Report against the CRE's production environment.
// The first time a DON's report is seen, the signatures will be fetched from the chain. It will be cached for later report parsing.
// VerifySignaturesWithConfig only needs to be called if SkipSignatureVerification was used with ParseReportWithConfig.
// Note config.SkipSignatureVerification is ignored by this method since if it were true, signatures would not be verified at all.
func (r *Report) VerifySignaturesWithConfig(runtime Runtime, config ReportParseConfig) error {
	config.SkipSignatureVerification = false
	header, err := r.parseHeader()
	if err != nil {
		return err
	}

	var candidates []Environment
	for _, zone := range config.AcceptedZones {
		if zone.DonID == header.donID {
			candidates = append(candidates, zone.Environment)
		}
	}
	candidates = append(candidates, config.AcceptedEnvironments...)

	if len(candidates) == 0 {
		return fmt.Errorf("DON ID %d is not in accepted zones", header.donID)
	}

	var sigErr error
	var fetchFailures []error
	for _, env := range candidates {
		f, signers, err := fetchDONInfo(runtime, env, header.donID)
		if err != nil {
			readErr := fmt.Errorf(
				"could not read from chain %d contract %s: %w",
				env.ChainSelector,
				env.RegistryAddress,
				err,
			)
			fetchFailures = append(fetchFailures, readErr)
			continue
		}

		if sigErr = verifySigs(r.report, f, signers); sigErr == nil {
			return nil
		}
	}

	if len(fetchFailures) > 0 {
		return errors.Join(fetchFailures...)
	}

	return sigErr
}

// fetchDONInfo performs a two-step on-chain lookup to retrieve the fault-
// tolerance parameter (f) and the authorized signer addresses for a DON:
//
//  1. getDON(donID)           → DONInfo containing f and nodeP2PIds
//  2. getNodesByP2PIds(ids)   → NodeInfo[] containing signer bytes32 per node
//
// Results are cached per chain+DON.
func fetchDONInfo(runtime Runtime, env Environment, donID uint32) (int, map[common.Address]uint32, error) {
	cacheKey := fmt.Sprintf("%d:%d", env.ChainSelector, donID)
	if cached, ok := keyCache.Load(cacheKey); ok {
		info := cached.(donInfo)
		return info.f, info.signers, nil
	}

	registryAddrHex := strings.TrimPrefix(env.RegistryAddress, "0x")
	registryAddrHex = strings.TrimPrefix(registryAddrHex, "0X")
	registryAddr, err := hex.DecodeString(registryAddrHex)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid registry address %q: %w", env.RegistryAddress, err)
	}

	capID := "evm:ChainSelector:" + strconv.FormatUint(env.ChainSelector, 10) + "@1.0.0"

	// keccak256("getDON(uint32)")[0:4] = 0x23537405
	var donIDPadded [32]byte
	binary.BigEndian.PutUint32(donIDPadded[28:], donID)
	getDONCallData := append([]byte{0x23, 0x53, 0x74, 0x05}, donIDPadded[:]...)

	getDONABI, err := callContract(runtime, capID, registryAddr, getDONCallData)
	if err != nil {
		return 0, nil, err
	}

	// ABI layout for getDON (tuple with outer pointer at byte 0):
	//   slot 0  (bytes   0-31):  outer pointer (=32)
	//   slot 1  (bytes  32-63):  id            (uint32)
	//   slot 2  (bytes  64-95):  configCount   (uint32)
	//   slot 3  (bytes  96-127): f             (uint8, zero-padded to 32)
	//   slot 4  (bytes 128-159): isPublic      (bool)
	//   slot 5  (bytes 160-191): acceptsWorkflows (bool)
	//   slot 6  (bytes 192-223): ptr[nodeP2PIds]  (bytes32[])
	//   slot 7  (bytes 224-255): ptr[donFamilies] (string[])
	//   slot 8  (bytes 256-287): ptr[name]        (string)
	//   slot 9  (bytes 288-319): ptr[config]      (bytes)
	//   slot 10 (bytes 320-351): ptr[capabilityConfigurations] (tuple[])
	if len(getDONABI) < 224 {
		return 0, nil, fmt.Errorf("getDON ABI response too short: %d bytes", len(getDONABI))
	}

	f := int(new(big.Int).SetBytes(getDONABI[96:128]).Int64())

	// Read nodeP2PIds from slot 6.
	// Pointer is tuple-relative; tuple starts at byte 32.
	const tupleStart = 32
	nodeP2PIdsPtr := int(new(big.Int).SetBytes(getDONABI[192:224]).Int64())
	nodeCountOff := tupleStart + nodeP2PIdsPtr
	if nodeCountOff+32 > len(getDONABI) {
		return 0, nil, fmt.Errorf("getDON ABI: nodeP2PIds pointer out of range")
	}
	nodeCount := int(new(big.Int).SetBytes(getDONABI[nodeCountOff : nodeCountOff+32]).Int64())
	if nodeCountOff+32+nodeCount*32 > len(getDONABI) {
		return 0, nil, fmt.Errorf("getDON ABI: nodeP2PIds data out of range")
	}
	nodeP2PIds := make([][]byte, nodeCount)
	for i := 0; i < nodeCount; i++ {
		start := nodeCountOff + 32 + i*32
		id := make([]byte, 32)
		copy(id, getDONABI[start:start+32])
		nodeP2PIds[i] = id
	}

	if nodeCount == 0 {
		info := donInfo{f: f, signers: nil}
		keyCache.Store(cacheKey, info)
		return f, nil, nil
	}

	// keccak256("getNodesByP2PIds(bytes32[])")[0:4] = 0x05a51966
	//
	// ABI-encode bytes32[]: [ptr=32][count][id0]...[idN]
	var p2pIdsABI []byte
	p2pIdsABI = append(p2pIdsABI, padUint256(32)...)
	p2pIdsABI = append(p2pIdsABI, padUint256(uint64(nodeCount))...)
	for _, id := range nodeP2PIds {
		p2pIdsABI = append(p2pIdsABI, id...)
	}
	getNodesCallData := append([]byte{0x05, 0xa5, 0x19, 0x66}, p2pIdsABI...)

	getNodesABI, err := callContract(runtime, capID, registryAddr, getNodesCallData)
	if err != nil {
		return 0, nil, err
	}

	// ABI layout for getNodesByP2PIds: NodeInfo[] (tuple array).
	// Each NodeInfo tuple head (9 slots × 32 bytes = 288 bytes per node):
	//   slot 0: nodeOperatorId (uint32)
	//   slot 1: configCount    (uint32)
	//   slot 2: workflowDONId  (uint32)
	//   slot 3: signer         (bytes32)  ← address left-aligned (first 20 bytes, rest zero)
	//   slot 4: p2pId          (bytes32)
	//   slot 5: encryptionPublicKey (bytes32)
	//   slot 6: csaKey         (bytes32)
	//   slot 7: ptr[capabilityIds]    (dynamic)
	//   slot 8: ptr[capabilitiesDONIds] (dynamic)
	//
	// The outer array encoding: [outer-ptr=32][count][tuple0-head][tuple1-head]...
	if len(getNodesABI) < 64 {
		return 0, nil, fmt.Errorf("getNodesByP2PIds ABI response too short: %d bytes", len(getNodesABI))
	}

	// The outer pointer (slot 0) points to the array data relative to byte 0.
	outerPtr := int(new(big.Int).SetBytes(getNodesABI[0:32]).Int64())
	if outerPtr+32 > len(getNodesABI) {
		return 0, nil, fmt.Errorf("getNodesByP2PIds ABI: outer pointer out of range")
	}
	returnedCount := int(new(big.Int).SetBytes(getNodesABI[outerPtr : outerPtr+32]).Int64())

	const nodeTupleHeadSize = 288 // 9 slots × 32 bytes
	signers := make(map[common.Address]uint32, returnedCount)
	for i := 0; i < returnedCount; i++ {
		// Each tuple in a dynamic array is referenced via a per-element pointer
		// stored at outerPtr+32+i*32, which is tuple-relative to outerPtr+32.
		elemPtrOff := outerPtr + 32 + i*32
		if elemPtrOff+32 > len(getNodesABI) {
			break
		}
		elemPtr := int(new(big.Int).SetBytes(getNodesABI[elemPtrOff : elemPtrOff+32]).Int64())
		tupleBase := outerPtr + 32 + elemPtr
		if tupleBase+nodeTupleHeadSize > len(getNodesABI) {
			break
		}
		// slot 0 of the tuple = nodeOperatorId (uint32).
		nodeOperatorId := uint32(new(big.Int).SetBytes(getNodesABI[tupleBase : tupleBase+32]).Uint64())
		// slot 3 of the tuple = signer bytes32; address is the first 20 bytes (left-aligned).
		signerSlot := tupleBase + 3*32
		addr := common.BytesToAddress(getNodesABI[signerSlot : signerSlot+20])
		signers[addr] = nodeOperatorId
	}

	info := donInfo{f: f, signers: signers}

	keyCache.Store(cacheKey, info)
	return f, signers, nil
}

// callContract sends a CallContractRequest via the EVM capability and returns
// the raw ABI-encoded response bytes.
func callContract(runtime Runtime, capID string, registryAddr []byte, callData []byte) ([]byte, error) {
	var callMsgBytes []byte
	callMsgBytes = protowire.AppendTag(callMsgBytes, 2, protowire.BytesType)
	callMsgBytes = protowire.AppendBytes(callMsgBytes, registryAddr)
	callMsgBytes = protowire.AppendTag(callMsgBytes, 3, protowire.BytesType)
	callMsgBytes = protowire.AppendBytes(callMsgBytes, callData)

	// BigInt for FinalizedBlockNumber (-3): {abs_val: [3], sign: int64(-1)}
	var bigIntBytes []byte
	bigIntBytes = protowire.AppendTag(bigIntBytes, 1, protowire.BytesType)
	bigIntBytes = protowire.AppendBytes(bigIntBytes, []byte{0x03})
	bigIntBytes = protowire.AppendTag(bigIntBytes, 2, protowire.VarintType)
	bigIntBytes = protowire.AppendVarint(bigIntBytes, 0xFFFFFFFFFFFFFFFF) // int64(-1)

	// CallContractRequest: field 2 (block_number) before field 1 (call).
	var reqBytes []byte
	reqBytes = protowire.AppendTag(reqBytes, 2, protowire.BytesType)
	reqBytes = protowire.AppendBytes(reqBytes, bigIntBytes)
	reqBytes = protowire.AppendTag(reqBytes, 1, protowire.BytesType)
	reqBytes = protowire.AppendBytes(reqBytes, callMsgBytes)

	payload := &anypb.Any{
		TypeUrl: "type.googleapis.com/capabilities.blockchain.evm.v1alpha.CallContractRequest",
		Value:   reqBytes,
	}

	resp, err := runtime.CallCapability(&sdk.CapabilityRequest{
		Id:      capID,
		Payload: payload,
		Method:  "CallContract",
	}).Await()
	if err != nil {
		return nil, fmt.Errorf("EVM capability call failed: %w", err)
	}

	switch r := resp.Response.(type) {
	case *sdk.CapabilityResponse_Error:
		return nil, errors.New(r.Error)
	case *sdk.CapabilityResponse_Payload:
		abiData, err := decodeCallContractReplyData(r.Payload.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CallContractReply: %w", err)
		}
		return abiData, nil
	default:
		return nil, fmt.Errorf("unexpected EVM capability response type")
	}
}

// padUint256 encodes a uint64 as a big-endian 32-byte value.
func padUint256(v uint64) []byte {
	b := make([]byte, 32)
	binary.BigEndian.PutUint64(b[24:], v)
	return b
}

// decodeCallContractReplyData extracts the ABI-encoded data bytes from the
// proto-encoded CallContractReply (field 1 = data bytes).
func decodeCallContractReplyData(protoBytes []byte) ([]byte, error) {
	for len(protoBytes) > 0 {
		fieldNum, wireType, n := protowire.ConsumeTag(protoBytes)
		if n < 0 {
			return nil, fmt.Errorf("malformed CallContractReply proto")
		}
		protoBytes = protoBytes[n:]

		if fieldNum == 1 && wireType == protowire.BytesType {
			v, m := protowire.ConsumeBytes(protoBytes)
			if m < 0 {
				return nil, fmt.Errorf("malformed data field in CallContractReply")
			}
			return v, nil
		}

		// Skip unknown fields.
		var skipLen int
		switch wireType {
		case protowire.VarintType:
			_, skipLen = protowire.ConsumeVarint(protoBytes)
		case protowire.Fixed32Type:
			_, skipLen = protowire.ConsumeFixed32(protoBytes)
		case protowire.Fixed64Type:
			_, skipLen = protowire.ConsumeFixed64(protoBytes)
		case protowire.BytesType:
			_, skipLen = protowire.ConsumeBytes(protoBytes)
		default:
			return nil, fmt.Errorf("unsupported wire type %v in CallContractReply", wireType)
		}
		if skipLen < 0 {
			return nil, fmt.Errorf("failed to skip field in CallContractReply")
		}
		protoBytes = protoBytes[skipLen:]
	}
	return nil, fmt.Errorf("data field not found in CallContractReply")
}

// verifySigs validates the signatures on a report and updates report.Sigs to
// contain only the accepted AttributedSignatures (with SignerId set).
//
// When an authorized-signer list is available from the on-chain registry, the
// first f+1 valid signatures from that list are accepted regardless of how
// many total signatures are present. When no signer list is available (e.g.
// an older contract version), exactly f+1 valid unique signatures are required.
func verifySigs(report *sdk.ReportResponse, f int, authorizedSigners map[common.Address]uint32) error {
	required := f + 1
	sigs := report.GetSigs()

	if len(sigs) < required {
		return fmt.Errorf("%w: got %d, need at least %d (f+1)", ErrWrongSignatureCount, len(sigs), required)
	}

	reportHash := crypto.Keccak256Hash(
		append(crypto.Keccak256(report.GetRawReport()), report.GetReportContext()...),
	)

	seen := make(map[common.Address]bool, len(sigs))
	accepted := make([]*sdk.AttributedSignature, 0, required)
	var skipErrs []error

	for i, attrSig := range sigs {
		if len(accepted) == required {
			break
		}

		sigBytes := make([]byte, len(attrSig.GetSignature()))
		copy(sigBytes, attrSig.GetSignature())

		if len(sigBytes) != 65 {
			skipErrs = append(skipErrs, fmt.Errorf("index %d: %w: has %d bytes, expected 65",
				i, ErrParseSignature, len(sigBytes)))
			continue
		}

		// Normalise legacy Ethereum v values (27/28 → 0/1).
		if sigBytes[64] == 27 || sigBytes[64] == 28 {
			sigBytes[64] -= 27
		}

		pubKey, err := crypto.SigToPub(reportHash.Bytes(), sigBytes)
		if err != nil {
			skipErrs = append(skipErrs, fmt.Errorf("index %d: %w: %s", i, ErrRecoverSigner, err))
			continue
		}

		signer := crypto.PubkeyToAddress(*pubKey)
		if seen[signer] {
			skipErrs = append(skipErrs, fmt.Errorf("index %d: %w: %s", i, ErrDuplicateSigner, signer.Hex()))
			continue
		}
		seen[signer] = true

		nodeOperatorId, ok := authorizedSigners[signer]
		if !ok {
			skipErrs = append(skipErrs, fmt.Errorf("index %d: %w: %s", i, ErrUnknownSigner, signer.Hex()))
			continue
		}
		attrSig.SignerId = nodeOperatorId

		accepted = append(accepted, attrSig)
	}

	if len(accepted) < required {
		if len(skipErrs) > 0 {
			return errors.Join(skipErrs...)
		}
		return fmt.Errorf("%w: only %d valid, need %d (f+1)", ErrWrongSignatureCount, len(accepted), required)
	}

	// Replace Sigs with only the accepted f+1 entries.
	report.Sigs = accepted
	return nil
}
