package cre

import (
	"crypto/sha256"
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
	// solanaSigners maps Solana signer address → nodeOperatorId, read from the consensus capability's OCR3 config.
	// solanaSignersErr is set when the DON does not publish Solana signers; it only fails Solana report verification.
	solanaSigners    map[common.Address]uint32
	solanaSignersErr error
}

// signersFor returns the authorized signers for reports produced by the given encoder.
func (d donInfo) signersFor(encoder string) (map[common.Address]uint32, error) {
	if encoder == EncoderSolana {
		return d.solanaSigners, d.solanaSignersErr
	}
	return d.signers, nil
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
	// Encoder is the encoder name the report was generated with (ReportRequest.EncoderName).
	// An empty value defaults to EncoderEVM. Only EncoderEVM and EncoderSolana reports can currently be parsed and
	// verified; any other value returns ErrUnsupportedEncoder.
	Encoder string
}

const (
	// EncoderEVM is the encoder name for reports signed with the EVM (ECDSA secp256k1) key bundle.
	EncoderEVM = "evm"
	// EncoderSolana is the encoder name for reports signed with the Solana (ECDSA secp256k1) key bundle.
	EncoderSolana = "solana"
)

// encoder returns the normalized encoder name, defaulting to EncoderEVM.
func (c ReportParseConfig) encoder() (string, error) {
	switch encoder := strings.ToLower(c.Encoder); encoder {
	case "":
		return EncoderEVM, nil
	case EncoderEVM, EncoderSolana:
		return encoder, nil
	default:
		return "", fmt.Errorf("%w %q: only %q and %q reports can be parsed and verified", ErrUnsupportedEncoder, c.Encoder, EncoderEVM, EncoderSolana)
	}
}

var defaultVerificationConfig = ReportParseConfig{AcceptedEnvironments: []Environment{ProductionEnvironment()}}

// ParseReport parses a CRE report and verifies it against the production CRE environment.
// Only reports generated with the "evm" encoder are supported; use the WithConfig variant and ReportParseConfig.Encoder for other encoders.
// The first time a DON's report is seen, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReport(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte) (*Report, error) {
	return ParseReportWithConfig(runtime, rawReport, signatures, reportContext, defaultVerificationConfig)
}

// ParseReportWithConfig parses a CRE report and verifies it as specified by the config.
// The first time a DON's report is seen, and SkipSignatureVerification is false, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReportWithConfig(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte, config ReportParseConfig) (*Report, error) {
	if _, err := config.encoder(); err != nil {
		return nil, err
	}

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
// Only reports generated with the "evm" encoder are supported; use the WithConfig variant and ReportParseConfig.Encoder for other encoders.
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
	encoder, err := config.encoder()
	if err != nil {
		return err
	}

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

	reportHash := hashReport(encoder, r.report)

	var sigErr error
	var fetchFailures []error
	for _, env := range candidates {
		info, err := fetchDONInfo(runtime, env, header.donID)
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

		signers, err := info.signersFor(encoder)
		if err != nil {
			fetchFailures = append(fetchFailures, fmt.Errorf(
				"no %s signers for DON %d on chain %d contract %s: %w",
				encoder,
				header.donID,
				env.ChainSelector,
				env.RegistryAddress,
				err,
			))
			continue
		}

		if sigErr = verifySigs(r.report, info.f, signers, reportHash); sigErr == nil {
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
//  1. getDON(donID)           → DONInfo containing f, nodeP2PIds and capabilityConfigurations
//  2. getNodesByP2PIds(ids)   → NodeInfo[] containing signer bytes32 per node
//
// Solana signers are read from the consensus capability's OCR3 config in capabilityConfigurations.
// Results are cached per chain+DON.
func fetchDONInfo(runtime Runtime, env Environment, donID uint32) (donInfo, error) {
	cacheKey := fmt.Sprintf("%d:%d", env.ChainSelector, donID)
	if cached, ok := keyCache.Load(cacheKey); ok {
		return cached.(donInfo), nil
	}

	registryAddrHex := strings.TrimPrefix(env.RegistryAddress, "0x")
	registryAddrHex = strings.TrimPrefix(registryAddrHex, "0X")
	registryAddr, err := hex.DecodeString(registryAddrHex)
	if err != nil {
		return donInfo{}, fmt.Errorf("invalid registry address %q: %w", env.RegistryAddress, err)
	}

	capID := "evm:ChainSelector:" + strconv.FormatUint(env.ChainSelector, 10) + "@1.0.0"

	// keccak256("getDON(uint32)")[0:4] = 0x23537405
	var donIDPadded [32]byte
	binary.BigEndian.PutUint32(donIDPadded[28:], donID)
	getDONCallData := append([]byte{0x23, 0x53, 0x74, 0x05}, donIDPadded[:]...)

	getDONABI, err := callContract(runtime, capID, registryAddr, getDONCallData)
	if err != nil {
		return donInfo{}, err
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
		return donInfo{}, fmt.Errorf("getDON ABI response too short: %d bytes", len(getDONABI))
	}

	f := int(new(big.Int).SetBytes(getDONABI[96:128]).Int64())

	// Read nodeP2PIds from slot 6.
	// Pointer is tuple-relative; tuple starts at byte 32.
	const tupleStart = 32
	nodeP2PIdsPtr := int(new(big.Int).SetBytes(getDONABI[192:224]).Int64())
	nodeCountOff := tupleStart + nodeP2PIdsPtr
	if nodeCountOff+32 > len(getDONABI) {
		return donInfo{}, fmt.Errorf("getDON ABI: nodeP2PIds pointer out of range")
	}
	nodeCount := int(new(big.Int).SetBytes(getDONABI[nodeCountOff : nodeCountOff+32]).Int64())
	if nodeCountOff+32+nodeCount*32 > len(getDONABI) {
		return donInfo{}, fmt.Errorf("getDON ABI: nodeP2PIds data out of range")
	}
	nodeP2PIds := make([][]byte, nodeCount)
	for i := 0; i < nodeCount; i++ {
		start := nodeCountOff + 32 + i*32
		id := make([]byte, 32)
		copy(id, getDONABI[start:start+32])
		nodeP2PIds[i] = id
	}

	if nodeCount == 0 {
		info := donInfo{f: f}
		keyCache.Store(cacheKey, info)
		return info, nil
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
		return donInfo{}, err
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
		return donInfo{}, fmt.Errorf("getNodesByP2PIds ABI response too short: %d bytes", len(getNodesABI))
	}

	// The outer pointer (slot 0) points to the array data relative to byte 0.
	outerPtr := int(new(big.Int).SetBytes(getNodesABI[0:32]).Int64())
	if outerPtr+32 > len(getNodesABI) {
		return donInfo{}, fmt.Errorf("getNodesByP2PIds ABI: outer pointer out of range")
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
	info.solanaSigners, info.solanaSignersErr = solanaSignersFromDON(getDONABI, signers)

	keyCache.Store(cacheKey, info)
	return info, nil
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
func verifySigs(report *sdk.ReportResponse, f int, authorizedSigners map[common.Address]uint32, reportHash []byte) error {
	required := f + 1
	sigs := report.GetSigs()

	if len(sigs) < required {
		return fmt.Errorf("%w: got %d, need at least %d (f+1)", ErrWrongSignatureCount, len(sigs), required)
	}

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

		pubKey, err := crypto.SigToPub(reportHash, sigBytes)
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

// hashReport returns the digest the encoder's key bundle signs.
func hashReport(encoder string, report *sdk.ReportResponse) []byte {
	if encoder == EncoderSolana {
		return solanaReportHash(report)
	}
	return crypto.Keccak256(crypto.Keccak256(report.GetRawReport()), report.GetReportContext())
}

// solanaReportHash matches the Solana keyring and forwarder program:
// sha256(uint8(len(rawReport)) || rawReport || reportContext). The length prefix is a single, truncated byte.
func solanaReportHash(report *sdk.ReportResponse) []byte {
	h := sha256.New()
	h.Write([]byte{byte(len(report.GetRawReport()))})
	h.Write(report.GetRawReport())
	h.Write(report.GetReportContext())
	return h.Sum(nil)
}

const (
	// consensusCapabilityPrefix identifies the consensus capability in a DON's capabilityConfigurations.
	consensusCapabilityPrefix = "consensus@"
	// defaultOCR3ConfigKey is the OCR instance name used by single-instance capabilities.
	defaultOCR3ConfigKey = "__default__"

	// Chain type identifiers used in multichain onchain public keys (see corekeys.ChainType.Type).
	chainTypeEVM    = 1
	chainTypeSolana = 2

	// Protobuf field numbers from chainlink-common capabilities/pb/registry.proto.
	capabilityConfigOCR3ConfigsField = 9 // CapabilityConfig.ocr3_configs (map<string, OCR3Config>)
	mapEntryKeyField                 = 1
	mapEntryValueField               = 2
	ocr3ConfigSignersField           = 1 // OCR3Config.signers (repeated bytes)
)

// solanaSignersFromDON maps each Solana signer in the consensus capability's OCR3 config to its node operator.
// Every OCR3 signer is a multichain key containing the node's EVM and Solana keys; the EVM key is matched against
// the registry's node signers so that only signers belonging to the DON's nodes are accepted.
func solanaSignersFromDON(getDONABI []byte, evmSigners map[common.Address]uint32) (map[common.Address]uint32, error) {
	ocrSigners, err := consensusOCR3Signers(getDONABI)
	if err != nil {
		return nil, err
	}

	solanaSigners := make(map[common.Address]uint32, len(ocrSigners))
	for _, ocrSigner := range ocrSigners {
		keys, err := parseMultichainPublicKey(ocrSigner)
		if err != nil {
			return nil, err
		}

		evmKey, solanaKey := keys[chainTypeEVM], keys[chainTypeSolana]
		if len(evmKey) != common.AddressLength || len(solanaKey) != common.AddressLength {
			continue
		}

		nodeOperatorId, ok := evmSigners[common.BytesToAddress(evmKey)]
		if !ok {
			continue
		}
		solanaSigners[common.BytesToAddress(solanaKey)] = nodeOperatorId
	}

	if len(solanaSigners) == 0 {
		return nil, errors.New("consensus OCR3 config has no solana signers")
	}
	return solanaSigners, nil
}

// consensusOCR3Signers reads the signers of the consensus capability's default OCR3 config from
// slot 10 (capabilityConfigurations) of the getDON ABI response.
//
// ABI layout of capabilityConfigurations (tuple(string capabilityId, bytes config)[]):
//
//	[count][elem0-ptr]...[elemN-ptr] then per element: [ptr[capabilityId]][ptr[config]] + string/bytes data
//
// Element pointers are relative to the first element pointer; field pointers are relative to the element.
func consensusOCR3Signers(getDONABI []byte) ([][]byte, error) {
	const tupleStart = 32
	capConfigsPtr, err := readABIOffset(getDONABI, 320)
	if err != nil {
		return nil, fmt.Errorf("getDON ABI: capabilityConfigurations pointer: %w", err)
	}

	arrayStart := tupleStart + capConfigsPtr
	count, err := readABIOffset(getDONABI, arrayStart)
	if err != nil {
		return nil, fmt.Errorf("getDON ABI: capabilityConfigurations count: %w", err)
	}

	elemsStart := arrayStart + 32
	for i := 0; i < count; i++ {
		elemPtr, err := readABIOffset(getDONABI, elemsStart+i*32)
		if err != nil {
			return nil, fmt.Errorf("getDON ABI: capabilityConfigurations[%d]: %w", i, err)
		}

		elem := elemsStart + elemPtr
		capabilityID, err := readABIDynamicField(getDONABI, elem, 0)
		if err != nil {
			return nil, fmt.Errorf("getDON ABI: capabilityConfigurations[%d].capabilityId: %w", i, err)
		}
		if !strings.HasPrefix(string(capabilityID), consensusCapabilityPrefix) {
			continue
		}

		config, err := readABIDynamicField(getDONABI, elem, 1)
		if err != nil {
			return nil, fmt.Errorf("getDON ABI: capabilityConfigurations[%d].config: %w", i, err)
		}
		return ocr3SignersFromCapabilityConfig(config)
	}

	return nil, errors.New("consensus capability is not configured on the DON")
}

// readABIOffset reads the 32-byte word at off as a non-negative int that fits within data.
func readABIOffset(data []byte, off int) (int, error) {
	if off < 0 || off+32 > len(data) {
		return 0, fmt.Errorf("offset %d out of range", off)
	}
	v := new(big.Int).SetBytes(data[off : off+32])
	if !v.IsInt64() || v.Int64() > int64(len(data)) {
		return 0, fmt.Errorf("value at offset %d out of range", off)
	}
	return int(v.Int64()), nil
}

// readABIDynamicField reads the string/bytes value referenced by the given head slot of the tuple at tupleBase.
func readABIDynamicField(data []byte, tupleBase, slot int) ([]byte, error) {
	ptr, err := readABIOffset(data, tupleBase+slot*32)
	if err != nil {
		return nil, err
	}
	length, err := readABIOffset(data, tupleBase+ptr)
	if err != nil {
		return nil, err
	}
	start := tupleBase + ptr + 32
	if start+length > len(data) {
		return nil, fmt.Errorf("data at offset %d out of range", start)
	}
	return data[start : start+length], nil
}

// ocr3SignersFromCapabilityConfig extracts ocr3_configs["__default__"].signers from a proto-encoded CapabilityConfig.
func ocr3SignersFromCapabilityConfig(config []byte) ([][]byte, error) {
	var signers [][]byte
	found := false
	err := forEachProtoBytesField(config, func(num protowire.Number, entry []byte) error {
		if num != capabilityConfigOCR3ConfigsField || found {
			return nil
		}

		var key string
		var ocr3Config []byte
		if err := forEachProtoBytesField(entry, func(num protowire.Number, v []byte) error {
			switch num {
			case mapEntryKeyField:
				key = string(v)
			case mapEntryValueField:
				ocr3Config = v
			}
			return nil
		}); err != nil {
			return err
		}
		if key != defaultOCR3ConfigKey {
			return nil
		}

		found = true
		return forEachProtoBytesField(ocr3Config, func(num protowire.Number, v []byte) error {
			if num == ocr3ConfigSignersField {
				signers = append(signers, v)
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("malformed consensus capability config: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("consensus capability config has no %q OCR3 config", defaultOCR3ConfigKey)
	}
	return signers, nil
}

// forEachProtoBytesField calls fn for every length-delimited field in a proto message, skipping all other fields.
func forEachProtoBytesField(msg []byte, fn func(protowire.Number, []byte) error) error {
	for len(msg) > 0 {
		num, typ, n := protowire.ConsumeTag(msg)
		if n < 0 {
			return protowire.ParseError(n)
		}
		msg = msg[n:]

		if typ != protowire.BytesType {
			n = protowire.ConsumeFieldValue(num, typ, msg)
			if n < 0 {
				return protowire.ParseError(n)
			}
			msg = msg[n:]
			continue
		}

		v, n := protowire.ConsumeBytes(msg)
		if n < 0 {
			return protowire.ParseError(n)
		}
		if err := fn(num, v); err != nil {
			return err
		}
		msg = msg[n:]
	}
	return nil
}

// parseMultichainPublicKey splits an OCR3 multichain onchain public key into its per-chain keys.
// Encoding: repeated [chainType uint8][length uint16 little-endian][key].
func parseMultichainPublicKey(data []byte) (map[uint8][]byte, error) {
	keys := make(map[uint8][]byte)
	for len(data) > 0 {
		if len(data) < 3 {
			return nil, errors.New("malformed multichain public key: truncated header")
		}
		chainType := data[0]
		length := int(binary.LittleEndian.Uint16(data[1:3]))
		if len(data) < 3+length {
			return nil, errors.New("malformed multichain public key: truncated key")
		}
		keys[chainType] = data[3 : 3+length]
		data = data[3+length:]
	}
	return keys, nil
}
