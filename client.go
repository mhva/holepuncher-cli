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
	"reflect"
	"strings"

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
		log.WithFields(log.Fields{
			"rpc":   c.reflectRPCName(m),
			"cause": err,
		}).Error("I/O error during RPC")
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"rpc":    c.reflectRPCName(m),
			"cause":  err,
			"status": response.StatusCode,
		}).Error("I/O error during RPC")
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		// Bail out on text/plain failures by logging the error and returning.
		// Protobuf errors will be passed back to the caller.
		if strings.ToLower(response.Header.Get("Content-Type")) == "text/plain" ||
			strings.HasPrefix(response.Header.Get("Content-Type"), "text/plain;") {
			cause := string(body)
			log.WithFields(log.Fields{
				"rpc":   c.reflectRPCName(m),
				"cause": cause,
			}).Error("Early RPC failure")
			return nil, errors.New(cause)
		}
	}

	responseMsg := &protoapi.Response{}
	if err = c.proto.ReadMessage(responseMsg, body); err != nil {
		log.WithFields(log.Fields{
			"rpc":   c.reflectRPCName(m),
			"cause": err,
		}).Error("RPC return value could not be decoded")
		return nil, err
	}
	return responseMsg, nil
}

func (c *holepuncherClient) reflectRPCName(m *protoapi.Request) string {
	if msgType := reflect.TypeOf(m.R); msgType != nil && msgType.Kind() == reflect.Ptr {
		return msgType.Elem().PkgPath() + "." + msgType.Elem().Name()
	}
	return "nil"
}

func decodeProtobufKey(key string, keyName string) ([]byte, error) {
	raw, err := hex.DecodeString(key)
	if err != nil {
		return nil, logConfigurationError("invalid key hex data", log.Fields{"key": keyName})
	}
	return raw, nil
}
