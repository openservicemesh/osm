package events

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
)

func TestPubSubEvents(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		register      announcements.AnnouncementType
		publish       PubSubMessage
		expectMessage bool
	}{
		{
			register: announcements.BackpressureAdded,
			publish: PubSubMessage{
				AnnouncementType: announcements.ConfigMapAdded,
				NewObj:           struct{}{},
				OldObj:           nil,
			},
			expectMessage: false,
		},
		{
			register: announcements.BackpressureAdded,
			publish: PubSubMessage{
				AnnouncementType: announcements.BackpressureAdded,
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

	subChannel := GetPubSubInstance().Subscribe(announcements.BackpressureUpdated)

	// publish something
	GetPubSubInstance().Publish(PubSubMessage{
		AnnouncementType: announcements.BackpressureUpdated,
	})

	// make sure channel is drained and closed
	GetPubSubInstance().Unsub(subChannel)

	// Channel has to have been already emptied and closed
	_, ok := <-subChannel
	assert.False(ok)
}
