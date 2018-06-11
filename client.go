package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"protoapi"
	"protocore"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type aHolepuncherClient interface {
	DoRequest(m *protoapi.Request) (*protoapi.Response, error)
}

type holepuncherClient struct {
	client  *http.Client
	proto   *protocore.Proto
	options *programOptions
}

func newHolepuncherClient(
	options *programOptions,
	client ...*http.Client,
) (*holepuncherClient, error) {
	var httpClient *http.Client
	if len(client) > 0 {
		httpClient = client[0]
	} else {
		httpClient = http.DefaultClient
	}

	if len(options.ProtobufClient.ServerKey) == 0 {
		return nil, logConfigurationError("client_protobuf.server_key is empty or missing")
	}
	if len(options.ProtobufClient.PeerKey) == 0 {
		return nil, logConfigurationError("client_protobuf.peer_key server key is empty or missing")
	}

	srvKey, err := decodeProtobufKey(options.ProtobufClient.ServerKey, "client_protobuf.server_key")
	if err != nil {
		return nil, err
	}
	peerKey, err := decodeProtobufKey(options.ProtobufClient.PeerKey, "client_protobuf.peer_key")
	if err != nil {
		return nil, err
	}

	return &holepuncherClient{
		client:  httpClient,
		proto:   protocore.NewProto(peerKey, srvKey),
		options: options,
	}, nil
}

func (c *holepuncherClient) DoRequest(m *protoapi.Request) (*protoapi.Response, error) {
	var requestURL string
	var payload bytes.Buffer
	if err := c.proto.WriteMessage(&payload, m); err != nil {
		return nil, err
	}

	payloadB64 := base64.RawStdEncoding.EncodeToString(payload.Bytes())
	prefix := c.options.Runtime.ServerAddress
	if prefix[len(prefix)-1] != '/' {
		requestURL = fmt.Sprintf("%s/proto/%s", prefix, payloadB64)
	} else {
		requestURL = fmt.Sprintf("%sproto/%s", prefix, payloadB64)
	}

	response, err := c.client.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// HTTP 418 (I'm teapot) is used to indicate protobuf error, we pass it
	// back to the caller like a normal response.
	if (response.StatusCode < 200 || response.StatusCode > 299) &&
		response.StatusCode != http.StatusTeapot {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errFmt := "server returned an error (%d); couldn't retrieve error because: %s"
			return nil, errors.Errorf(errFmt, response.StatusCode, err.Error())
		}
		errFmt := "server returned error (%d): %s"
		return nil, errors.Errorf(errFmt, response.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Errorf("unable to read response body: %s", err.Error())
	}

	responseMsg := &protoapi.Response{}
	if err = c.proto.ReadMessage(responseMsg, body); err != nil {
		return nil, errors.Errorf("unable to decode protobuf response: %s", err.Error())
	}
	return responseMsg, nil
}

func decodeProtobufKey(key string, keyName string) ([]byte, error) {
	raw, err := hex.DecodeString(key)
	if err != nil {
		return nil, logConfigurationError("invalid key hex data", log.Fields{"key": keyName})
	}
	return raw, nil
}
