package guac

type ChannelManagement struct {
	ChannelList       map[string][]chan int
	RequestPolicyFunc func(string, string) []string
}

func NewChannelManagement() *ChannelManagement {
	return &ChannelManagement{
		ChannelList: map[string][]chan int{},
	}
}

func (c *ChannelManagement) Add(key string, ch chan int) error {
	if _, ok := c.ChannelList[key]; !ok {
		c.ChannelList[key] = []chan int{}
	}
	c.ChannelList[key] = append(c.ChannelList[key], ch)
	return nil
}

func (c *ChannelManagement) BroadCast(key string, op int) error {
	if list, ok := c.ChannelList[key]; ok {
		for _, ch := range list {
			ch <- op
		}
	}
	return nil
}
