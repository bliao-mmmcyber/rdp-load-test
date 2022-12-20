package guac

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type ChannelManagement struct {
	ChannelList map[string]map[string]chan int
	mu          sync.Mutex
}

func NewChannelManagement() *ChannelManagement {
	return &ChannelManagement{
		ChannelList: map[string]map[string]chan int{},
	}
}

func (c *ChannelManagement) Add(key string, ID string, ch chan int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.ChannelList[key]; !ok {
		c.ChannelList[key] = map[string]chan int{}
	}
	c.ChannelList[key][ID] = ch
	return nil
}

func (c *ChannelManagement) Remove(appID string, userID string, ID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	logrus.Info("Remove channel")
	if appList, ok := c.ChannelList[appID]; ok {
		if appCH, ok := appList[ID]; ok {
			close(appCH)
			delete(appList, ID)
		}
	}
	if userList, ok := c.ChannelList[userID]; ok {
		delete(userList, ID)
	}
	return nil
}

func (c *ChannelManagement) BroadCast(key string, op int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if list, ok := c.ChannelList[key]; ok {
		for _, ch := range list {
			ch <- op
		}
	}
	return nil
}
