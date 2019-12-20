/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package route

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	diddoc "github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	mockdispatcher "github.com/hyperledger/aries-framework-go/pkg/internal/mock/didcomm/dispatcher"
	mockkms "github.com/hyperledger/aries-framework-go/pkg/internal/mock/kms"
	mockprovider "github.com/hyperledger/aries-framework-go/pkg/internal/mock/provider"
	mockstore "github.com/hyperledger/aries-framework-go/pkg/internal/mock/storage"
	mockvdri "github.com/hyperledger/aries-framework-go/pkg/internal/mock/vdri"
)

const (
	MYDID    = "myDID"
	THEIRDID = "theirDID"
)

type updateResult struct {
	action string
	result string
}

func TestServiceNew(t *testing.T) {
	t.Run("test new service - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider()})
		require.NoError(t, err)
		require.Equal(t, Coordination, svc.Name())
	})

	t.Run("test new service name - failure", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{
			StorageProviderValue: &mockstore.MockStoreProvider{
				ErrOpenStoreHandle: fmt.Errorf("error opening the store")}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "open route coordination store")
		require.Nil(t, svc)
	})
}

func TestServiceAccept(t *testing.T) {
	s := &Service{}

	require.Equal(t, true, s.Accept(RequestMsgType))
	require.Equal(t, true, s.Accept(GrantMsgType))
	require.Equal(t, true, s.Accept(KeylistUpdateMsgType))
	require.Equal(t, true, s.Accept(KeylistUpdateResponseMsgType))
	require.Equal(t, true, s.Accept(ForwardMsgType))
	require.Equal(t, false, s.Accept("unsupported msg type"))
}

func TestServiceHandleInbound(t *testing.T) {
	t.Run("test handle outbound ", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider()})
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(&service.DIDCommMsg{Header: &service.Header{
			ID: msgID,
		}})
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})
}

func TestServiceHandleOutbound(t *testing.T) {
	t.Run("test handle outbound ", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider()})
		require.NoError(t, err)

		err = svc.HandleOutbound(nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not implemented")
	})
}

func TestServiceRequestMsg(t *testing.T) {
	t.Run("test service handle inbound request msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(generateRequestMsgPayload(t, msgID))
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})

	t.Run("test service handle request msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msg := &service.DIDCommMsg{Payload: []byte("invalid json")}

		err = svc.handleRequest(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "route request message unmarshal")
	})

	t.Run("test service handle request msg - verify outbound message", func(t *testing.T) {
		endpoint := "ws://agent.example.com"
		svc, err := New(&mockprovider.Provider{
			StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:             &mockkms.CloseableKMS{},
			InboundEndpointValue: endpoint,
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{
				ValidateSend: func(msg interface{}, senderVerKey string, des *service.Destination) error {
					res, err := json.Marshal(msg)
					require.NoError(t, err)

					grant := &Grant{}
					err = json.Unmarshal(res, grant)
					require.NoError(t, err)

					require.Equal(t, endpoint, grant.Endpoint)
					require.Equal(t, 1, len(grant.RoutingKeys))

					return nil
				},
			},
		})
		require.NoError(t, err)

		msgID := randomID()

		err = svc.handleRequest(generateRequestMsgPayload(t, msgID))
		require.NoError(t, err)
	})
}

func TestServiceGrantMsg(t *testing.T) {
	t.Run("test service handle inbound grant msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(generateGrantMsgPayload(t, msgID))
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})

	t.Run("test service handle grant msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msg := &service.DIDCommMsg{Payload: []byte("invalid json")}

		err = svc.handleGrant(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "route grant message unmarshal")
	})
}

func TestServiceUpdateKeyListMsg(t *testing.T) {
	t.Run("test service handle inbound key list update msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(generateKeyUpdateListMsgPayload(t, msgID, []Update{{
			RecipientKey: "ABC",
			Action:       "add",
		}}))
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})

	t.Run("test service handle key list update msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msg := &service.DIDCommMsg{Payload: []byte("invalid json")}

		err = svc.handleKeylistUpdate(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "route key list update message unmarshal")
	})

	t.Run("test service handle request msg - verify outbound message", func(t *testing.T) {
		update := make(map[string]updateResult)
		update["ABC"] = updateResult{action: add, result: success}
		update["XYZ"] = updateResult{action: remove, result: serverError}
		update[""] = updateResult{action: add, result: success}

		svc, err := New(&mockprovider.Provider{
			StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:             &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{
				ValidateSend: func(msg interface{}, senderVerKey string, des *service.Destination) error {
					res, err := json.Marshal(msg)
					require.NoError(t, err)

					updateRes := &KeylistUpdateResponse{}
					err = json.Unmarshal(res, updateRes)
					require.NoError(t, err)

					require.Equal(t, len(update), len(updateRes.Updated))

					for _, v := range updateRes.Updated {
						require.Equal(t, update[v.RecipientKey].action, v.Action)
						require.Equal(t, update[v.RecipientKey].result, v.Result)
					}

					return nil
				},
			},
		})
		require.NoError(t, err)

		msgID := randomID()

		var updates []Update
		for k, v := range update {
			updates = append(updates, Update{
				RecipientKey: k,
				Action:       v.action,
			})
		}

		err = svc.handleKeylistUpdate(generateKeyUpdateListMsgPayload(t, msgID, updates))
		require.NoError(t, err)
	})
}

func TestServiceKeylistUpdateResponseMsg(t *testing.T) {
	t.Run("test service handle inbound key list update response msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(generateKeylistUpdateResponseMsgPayload(t, msgID, []UpdateResponse{{
			RecipientKey: "ABC",
			Action:       "add",
			Result:       success,
		}}))
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})

	t.Run("test service handle key list update response msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msg := &service.DIDCommMsg{Payload: []byte("invalid json")}

		err = svc.handleKeylistUpdateResponse(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "route keylist update response message unmarshal")
	})
}

func TestServiceForwardMsg(t *testing.T) {
	t.Run("test service handle inbound forward msg - success", func(t *testing.T) {
		to := randomID()
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		err = svc.routeStore.Put(to, []byte("did:example:123"))
		require.NoError(t, err)

		msgID := randomID()

		id, err := svc.HandleInbound(generateForwardMsgPayload(t, msgID, to, nil))
		require.NoError(t, err)
		require.Equal(t, msgID, id)
	})

	t.Run("test service handle forward msg - success", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		msg := &service.DIDCommMsg{Payload: []byte("invalid json")}

		err = svc.handleForward(msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "forward message unmarshal")
	})

	t.Run("test service handle forward msg - route key fetch fail", func(t *testing.T) {
		to := randomID()
		msgID := randomID()

		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{}})
		require.NoError(t, err)

		err = svc.handleForward(generateForwardMsgPayload(t, msgID, to, nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "route key fetch")
	})

	t.Run("test service handle forward msg - validate forward message content", func(t *testing.T) {
		to := randomID()
		msgID := randomID()

		content := "packed message destined to the recipient through router"
		msg := generateForwardMsgPayload(t, msgID, to, content)

		svc, err := New(&mockprovider.Provider{
			StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:             &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{
				ValidateForward: func(msg interface{}, des *service.Destination) error {
					require.Equal(t, content, msg)

					return nil
				},
			},
		})
		require.NoError(t, err)

		err = svc.routeStore.Put(dataKey(to), []byte("did:example:123"))
		require.NoError(t, err)

		err = svc.handleForward(msg)
		require.NoError(t, err)
	})
}

func TestSendRequest(t *testing.T) {
	myDID := creatDID(MYDID, "")
	theirDID := creatDID(THEIRDID, "theirdid.endpoint")

	svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
		KMSValue: &mockkms.CloseableKMS{},
		OutboundDispatcherValue: &mockdispatcher.MockOutbound{
			ValidateSend: func(msg interface{}, senderVerKey string, des *service.Destination) error {
				require.Equal(t, string(myDID.PublicKey[0].Value), senderVerKey)
				require.Equal(t, theirDID.Service[0].ServiceEndpoint, des.ServiceEndpoint)
				return nil
			}},
		VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
			ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
				if didID == MYDID {
					return myDID, nil
				}
				if didID == THEIRDID {
					return theirDID, nil
				}
				return nil, nil
			}}})
	require.NoError(t, err)

	reqID, err := svc.SendRequest(myDID.ID, theirDID.ID)
	require.NoError(t, err)
	require.NotEmpty(t, reqID)
}

func TestSendRequestNegative(t *testing.T) {
	t.Run("test error from resolve my did", func(t *testing.T) {
		myDID := creatDID(MYDID, "")
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{},
			VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
				ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
					if didID == MYDID {
						return nil, fmt.Errorf("error resolve myDID")
					}
					return nil, nil
				}}})
		require.NoError(t, err)

		_, err = svc.SendRequest(myDID.ID, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error resolve myDID")
	})

	t.Run("test error from lookup recipient keys for my did", func(t *testing.T) {
		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{},
			VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
				ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
					if didID == MYDID {
						return &diddoc.Doc{}, nil
					}
					return nil, nil
				}}})
		require.NoError(t, err)

		_, err = svc.SendRequest(MYDID, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "sender verification keys not found")
	})

	t.Run("test error from resolve their did", func(t *testing.T) {
		myDID := creatDID(MYDID, "")

		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{},
			VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
				ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
					if didID == MYDID {
						return myDID, nil
					}
					if didID == THEIRDID {
						return nil, fmt.Errorf("error resolve theirDID")
					}
					return nil, nil
				}}})
		require.NoError(t, err)

		_, err = svc.SendRequest(myDID.ID, THEIRDID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error resolve theirDID")
	})

	t.Run("test error from create destination from their did", func(t *testing.T) {
		myDID := creatDID(MYDID, "")

		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue:                &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{},
			VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
				ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
					if didID == MYDID {
						return myDID, nil
					}
					if didID == THEIRDID {
						return &diddoc.Doc{}, nil
					}
					return nil, nil
				}}})
		require.NoError(t, err)

		_, err = svc.SendRequest(myDID.ID, THEIRDID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "prepare destination from their did")
	})

	t.Run("test error from outbound send", func(t *testing.T) {
		myDID := creatDID(MYDID, "")
		theirDID := creatDID(THEIRDID, "theirdid.endpoint")

		svc, err := New(&mockprovider.Provider{StorageProviderValue: mockstore.NewMockStoreProvider(),
			KMSValue: &mockkms.CloseableKMS{},
			OutboundDispatcherValue: &mockdispatcher.MockOutbound{
				ValidateSend: func(msg interface{}, senderVerKey string, des *service.Destination) error {
					return fmt.Errorf("error send")
				}},
			VDRIRegistryValue: &mockvdri.MockVDRIRegistry{
				ResolveFunc: func(didID string, opts ...vdri.ResolveOpts) (doc *diddoc.Doc, err error) {
					if didID == MYDID {
						return myDID, nil
					}
					if didID == THEIRDID {
						return theirDID, nil
					}
					return nil, nil
				}}})
		require.NoError(t, err)

		_, err = svc.SendRequest(myDID.ID, theirDID.ID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error send")
	})
}

func creatDID(didID, serviceEndpoint string) *diddoc.Doc {
	didContext := "https://w3id.org/did/v1"
	creator := didID + "#key-1"
	keyType := "Ed25519VerificationKey2018"

	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	s := diddoc.Service{
		ID:              "did:example:123456789abcdefghi#did-communication",
		Type:            "did-communication",
		ServiceEndpoint: serviceEndpoint,
		RecipientKeys:   []string{creator},
		Priority:        0,
	}

	signingKey := diddoc.PublicKey{
		ID:         creator,
		Type:       keyType,
		Controller: didID,
		Value:      pubKey,
	}

	createdTime := time.Now()

	return &diddoc.Doc{
		Context:   []string{didContext},
		ID:        didID,
		PublicKey: []diddoc.PublicKey{signingKey},
		Service:   []diddoc.Service{s},
		Created:   &createdTime,
	}
}

func generateRequestMsgPayload(t *testing.T, id string) *service.DIDCommMsg {
	requestBytes, err := json.Marshal(&Request{
		Type: RequestMsgType,
		ID:   id,
	})
	require.NoError(t, err)

	didMsg, err := service.NewDIDCommMsg(requestBytes)
	require.NoError(t, err)

	return didMsg
}

func generateGrantMsgPayload(t *testing.T, id string) *service.DIDCommMsg {
	grantBytes, err := json.Marshal(&Grant{
		Type: GrantMsgType,
		ID:   id,
	})
	require.NoError(t, err)

	didMsg, err := service.NewDIDCommMsg(grantBytes)
	require.NoError(t, err)

	return didMsg
}

func generateKeyUpdateListMsgPayload(t *testing.T, id string, updates []Update) *service.DIDCommMsg {
	requestBytes, err := json.Marshal(&KeylistUpdate{
		Type:    KeylistUpdateMsgType,
		ID:      id,
		Updates: updates,
	})
	require.NoError(t, err)

	didMsg, err := service.NewDIDCommMsg(requestBytes)
	require.NoError(t, err)

	return didMsg
}

func generateKeylistUpdateResponseMsgPayload(t *testing.T, id string, updates []UpdateResponse) *service.DIDCommMsg {
	respBytes, err := json.Marshal(&KeylistUpdateResponse{
		Type:    KeylistUpdateResponseMsgType,
		ID:      id,
		Updated: updates,
	})
	require.NoError(t, err)

	didMsg, err := service.NewDIDCommMsg(respBytes)
	require.NoError(t, err)

	return didMsg
}

func generateForwardMsgPayload(t *testing.T, id, to string, msg interface{}) *service.DIDCommMsg {
	requestBytes, err := json.Marshal(&Forward{
		Type: ForwardMsgType,
		ID:   id,
		To:   to,
		Msg:  msg,
	})
	require.NoError(t, err)

	didMsg, err := service.NewDIDCommMsg(requestBytes)
	require.NoError(t, err)

	return didMsg
}

func randomID() string {
	return uuid.New().String()
}