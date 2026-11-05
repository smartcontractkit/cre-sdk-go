package cre

import (
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	// Encoder is the encoder name the report was generated with (ReportRequest.EncoderName).
	// An empty value defaults to EncoderEVM. Only EVM reports can currently be parsed and verified;
	// any other value returns ErrUnsupportedEncoder.
	Encoder string
}

// EncoderEVM is the encoder name for reports signed with the EVM (ECDSA secp256k1) key bundle.
const EncoderEVM = "evm"

func (c ReportParseConfig) validateEncoder() error {
	if c.Encoder == "" || strings.EqualFold(c.Encoder, EncoderEVM) {
		return nil
	}
	return fmt.Errorf("%w %q: only %q reports can be parsed and verified", ErrUnsupportedEncoder, c.Encoder, EncoderEVM)
}

var defaultVerificationConfig = ReportParseConfig{AcceptedEnvironments: []Environment{ProductionEnvironment()}}

// ParseReport parses a CRE report and verifies it against the production CRE environment.
// Only reports generated with the "evm" encoder are supported; see ReportParseConfig.Encoder.
// The first time a DON's report is seen, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReport(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte) (*Report, error) {
	return ParseReportWithConfig(runtime, rawReport, signatures, reportContext, defaultVerificationConfig)
}

// ParseReportWithConfig parses a CRE report and verifies it as specified by the config.
// The first time a DON's report is seen, and SkipSignatureVerification is false, the signatures will be fetched from chain. It will be cached for later report parsing.
func ParseReportWithConfig(runtime Runtime, rawReport []byte, signatures [][]byte, reportContext []byte, config ReportParseConfig) (*Report, error) {
	if err := config.validateEncoder(); err != nil {
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
// Only reports generated with the "evm" encoder are supported; see ReportParseConfig.Encoder.
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
	if err := config.validateEncoder(); err != nil {
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

//go:embed capreg_abi.json
var capabilitiesRegistryABIJSON string

// capabilitiesRegistryABI is parsed on first use so workflows that never verify reports don't pay for it.
var capabilitiesRegistryABI = sync.OnceValues(func() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(capabilitiesRegistryABIJSON))
})

// capabilitiesRegistryDONInfo mirrors CapabilitiesRegistry.DONInfo; ABI unpacking matches fields by name.
type capabilitiesRegistryDONInfo struct {
	Id                       uint32
	ConfigCount              uint32
	F                        uint8
	IsPublic                 bool
	AcceptsWorkflows         bool
	NodeP2PIds               [][32]byte
	DonFamilies              []string
	Name                     string
	Config                   []byte
	CapabilityConfigurations []struct {
		CapabilityId string
		Config       []byte
	}
}

// capabilitiesRegistryNodeInfo mirrors CapabilitiesRegistry.NodeInfo; ABI unpacking matches fields by name.
type capabilitiesRegistryNodeInfo struct {
	NodeOperatorId      uint32
	ConfigCount         uint32
	WorkflowDONId       uint32
	Signer              [32]byte
	P2pId               [32]byte
	EncryptionPublicKey [32]byte
	CsaKey              [32]byte
	CapabilityIds       []string
	CapabilitiesDONIds  []*big.Int
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

	registryABI, err := capabilitiesRegistryABI()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse capabilities registry ABI: %w", err)
	}

	capID := "evm:ChainSelector:" + strconv.FormatUint(env.ChainSelector, 10) + "@1.0.0"

	don, err := callRegistry[capabilitiesRegistryDONInfo](runtime, capID, registryAddr, registryABI, "getDON", donID)
	if err != nil {
		return 0, nil, err
	}
	f := int(don.F)

	if len(don.NodeP2PIds) == 0 {
		info := donInfo{f: f, signers: nil}
		keyCache.Store(cacheKey, info)
		return f, nil, nil
	}

	nodes, err := callRegistry[[]capabilitiesRegistryNodeInfo](runtime, capID, registryAddr, registryABI, "getNodesByP2PIds", don.NodeP2PIds)
	if err != nil {
		return 0, nil, err
	}

	signers := make(map[common.Address]uint32, len(nodes))
	for _, node := range nodes {
		// The signer is an address left-aligned in a bytes32.
		signers[common.BytesToAddress(node.Signer[:common.AddressLength])] = node.NodeOperatorId
	}

	info := donInfo{f: f, signers: signers}

	keyCache.Store(cacheKey, info)
	return f, signers, nil
}

// callRegistry ABI-encodes a call to the capabilities registry, sends it via the EVM capability,
// and unpacks its single return value into T.
func callRegistry[T any](runtime Runtime, capID string, registryAddr []byte, registryABI abi.ABI, method string, args ...any) (T, error) {
	var out T
	callData, err := registryABI.Pack(method, args...)
	if err != nil {
		return out, fmt.Errorf("failed to encode %s call: %w", method, err)
	}

	result, err := callContract(runtime, capID, registryAddr, callData)
	if err != nil {
		return out, err
	}

	values, err := registryABI.Unpack(method, result)
	if err != nil {
		return out, fmt.Errorf("failed to decode %s response: %w", method, err)
	}
	if len(values) != 1 {
		return out, fmt.Errorf("failed to decode %s response: expected 1 value, got %d", method, len(values))
	}

	value, ok := convertABIValue(reflect.ValueOf(values[0]), reflect.TypeOf(out))
	if !ok {
		return out, fmt.Errorf("failed to decode %s response: cannot convert %T to %T", method, values[0], out)
	}
	return value.Interface().(T), nil
}

// convertABIValue converts an ABI-generated struct (or slice of structs) to an equivalent named type.
// Struct conversions ignore tags, but slices of different element types must be converted element by element.
func convertABIValue(value reflect.Value, to reflect.Type) (reflect.Value, bool) {
	if value.Type().ConvertibleTo(to) {
		return value.Convert(to), true
	}
	if value.Kind() != reflect.Slice || to.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}

	out := reflect.MakeSlice(to, value.Len(), value.Len())
	for i := 0; i < value.Len(); i++ {
		elem, ok := convertABIValue(value.Index(i), to.Elem())
		if !ok {
			return reflect.Value{}, false
		}
		out.Index(i).Set(elem)
	}
	return out, true
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
