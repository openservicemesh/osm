package dispatcher

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
)

func TestPubSubEvents(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		register      AnnouncementType
		publish       PubSubMessage
		expectMessage bool
	}{
		{
			register: EndpointAdded,
			publish: PubSubMessage{
				AnnouncementType: ConfigMapAdded,
				NewObj:           struct{}{},
				OldObj:           nil,
			},
			expectMessage: false,
		},
		{
			register: EndpointAdded,
			publish: PubSubMessage{
				AnnouncementType: EndpointAdded,
				NewObj:           nil,
				OldObj:           "randomString",
			},
			expectMessage: true,
		},
	}

	for i := range testCases {
		subscribedChanel := GetPubSubInstance().Subscribe(testCases[i].register)
		GetPubSubInstance().Publish(testCases[i].publish)

		select {
		case psMesg := <-subscribedChanel:
			assert.True(testCases[i].expectMessage)

			psCast, ok := psMesg.(PubSubMessage)
			assert.True(ok)

			equal := reflect.DeepEqual(psCast, testCases[i].publish)
			assert.True(equal)

		case <-time.After(1 * time.Second):
			assert.False(testCases[i].expectMessage)
		}
	}
}

func TestPubSubClose(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subChannel := GetPubSubInstance().Subscribe(EndpointUpdated)

	// publish something
	GetPubSubInstance().Publish(PubSubMessage{
		AnnouncementType: EndpointUpdated,
	})

	// make sure channel is drained and closed
	GetPubSubInstance().Unsubscribe(subChannel)

	// Channel has to have been already emptied and closed
	_, ok := <-subChannel
	assert.False(ok)
}
