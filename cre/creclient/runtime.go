package creclient

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"google.golang.org/protobuf/proto"
)

func NewRuntime() cre.Runtime {
	return &sdkimpl.Runtime{RuntimeBase: newRuntime(sdk.Mode_MODE_DON)}
}

func newRuntime(mode sdk.Mode) sdkimpl.RuntimeBase {
	return sdkimpl.RuntimeBase{
		Mode:           mode,
		RuntimeHelpers: &runtimeHelper{idMap: map[int32]string{}, secrets: map[int32]*sdk.SecretResponses{}},
		Lggr:           slog.Default(),
	}
}

type runtimeHelper struct {
	idMap   map[int32]string
	secrets map[int32]*sdk.SecretResponses
}

var _ sdkimpl.RuntimeHelpers = (*runtimeHelper)(nil)

func (r *runtimeHelper) Call(request *sdk.CapabilityRequest) error {
	anyBody, err := proto.Marshal(request.Payload)
	if err != nil {
		return err
	}

	body, err := json.Marshal(capabilityRequest{
		CapabilityId: request.Id,
		Payload:      base64.StdEncoding.EncodeToString(anyBody),
		Method:       request.Method,
	})
	if err != nil {
		return err
	}
	reqJ, err := json.Marshal(&httpRequest{
		Body:     body,
		Workflow: "../workflow/main.go",
	})
	if err != nil {
		return err
	}
	res, err := httpPost("call", string(reqJ))
	if err != nil {
		return err
	}

	r.idMap[request.CallbackId] = string(res)
	return nil
}

func (r *runtimeHelper) Await(request *sdk.AwaitCapabilitiesRequest, _ uint64) (*sdk.AwaitCapabilitiesResponse, error) {
	ids := make([]string, len(request.Ids))
	for i, cid := range request.Ids {
		rid, ok := r.idMap[cid]
		if !ok {
			return nil, fmt.Errorf(`request "%d" not found`, cid)
		}
		ids[i] = rid
	}

	idStrs := strings.Join(ids, ",")
	body, err := httpPost("await", idStrs)
	if err != nil {
		return nil, err
	}

	response := &sdk.AwaitCapabilitiesResponse{
		Responses: map[int32]*sdk.CapabilityResponse{},
	}

	for i := 0; i < len(request.Ids); i++ {
		delete(r.idMap, request.Ids[i])
		respSize := int32(binary.LittleEndian.Uint32(body))
		respBody := body[4 : 4+respSize]

		res := &result{}
		if err = json.Unmarshal(respBody, res); err != nil {
			return nil, err
		}
		if res.IsError {
			return nil, errors.New(res.Result)
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.Trim(res.Result, `"`))
		if err != nil {
			return nil, err
		}

		cr := &sdk.CapabilityResponse{}
		if err = proto.Unmarshal(decoded, cr); err != nil {
			return nil, err
		}

		switch r := cr.Response.(type) {
		case *sdk.CapabilityResponse_Error:
			return nil, errors.New(r.Error)
		case *sdk.CapabilityResponse_Payload:
			response.Responses[request.Ids[i]] = cr
		default:
			return nil, errors.New("unknown capability response")
		}
	}

	return response, nil
}

func (r *runtimeHelper) GetSecrets(request *sdk.GetSecretsRequest, _ uint64) error {
	// Do we make them local or get them from the vault DON?
	// Let's keep it local for now.
	response := make([]*sdk.SecretResponse, len(request.Requests))

	for i, secretRequest := range request.Requests {
		secretResponse := &sdk.SecretResponse{}
		response[i] = secretResponse
		secretVar := fmt.Sprintf("%s__%s", strings.ToUpper(secretRequest.Namespace), strings.ToUpper(secretRequest.Id))
		if value, ok := os.LookupEnv(secretVar); ok {
			secretResponse.Response = &sdk.SecretResponse_Secret{Secret: &sdk.Secret{Value: value, Namespace: secretRequest.Namespace, Id: secretRequest.Id}}
		} else {
			secretResponse.Response = &sdk.SecretResponse_Error{Error: &sdk.SecretError{Error: fmt.Sprintf("secret %s not found", secretVar), Namespace: secretRequest.Namespace, Id: secretRequest.Id}}
		}
	}

	r.secrets[request.CallbackId] = &sdk.SecretResponses{Responses: response}
	return nil
}

func (r *runtimeHelper) AwaitSecrets(request *sdk.AwaitSecretsRequest, _ uint64) (*sdk.AwaitSecretsResponse, error) {
	response := &sdk.AwaitSecretsResponse{Responses: map[int32]*sdk.SecretResponses{}}
	for _, id := range request.Ids {
		secrets, ok := r.secrets[id]
		if !ok {
			return nil, fmt.Errorf("secrets for callback id %d not found", id)
		}
		response.Responses[id] = secrets
	}

	return response, nil
}

func (r *runtimeHelper) SwitchModes(_ sdk.Mode) {}

func (r *runtimeHelper) GetSource(_ sdk.Mode) rand.Source {
	return rand.NewSource(time.Now().UnixNano())
}

func (r *runtimeHelper) Now() time.Time {
	return time.Now()
}

func httpPost(suffix string, body string) ([]byte, error) {
	client := http.Client{Timeout: time.Minute * 5}
	resp, err := client.Post("http://localhost:8090/"+suffix, "application/json", strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}
