package utils

import (
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
)

type Gossip struct {
	Name                    string
	BindAddr                string
	BindPort                int
	SecretKey               []byte
	ProbeInterval           int
	SyncInterval            int
	RetransmitMult          int
	Ready                   bool
	NotifyJoinHandler       func(*memberlist.Node)
	NotifyLeaveHandler      func(*memberlist.Node)
	NotifyUpdateHandler     func(*memberlist.Node)
	InvalidatesHandler      func(memberlist.Broadcast) bool
	NodeMetaHandler         func() []byte
	NotifyMsgHandler        func([]byte)
	LocalStateHandler       func() []byte
	MergeRemoteStateHandler func([]byte)
	queue                   *memberlist.TransmitLimitedQueue
	members                 *memberlist.Memberlist
}

// Start gossip
func (g *Gossip) Start(members *string) error {
	c := memberlist.DefaultLocalConfig()

	c.Name = g.Name
	c.BindAddr = g.BindAddr
	c.BindPort = g.BindPort
	c.SecretKey = g.SecretKey
	c.ProbeInterval = time.Duration(g.ProbeInterval) * time.Second
	c.PushPullInterval = time.Duration(g.SyncInterval) * time.Second
	c.RetransmitMult = g.RetransmitMult
	c.Events = &event_delegate_impl{
		gossip: g,
	}
	c.Delegate = &delegate_impl{
		gossip: g,
	}

	if ml, err := memberlist.Create(c); err != nil {
		return err
	} else {
		g.members = ml
	}

	if members != nil && len(*members) > 0 {
		parts := strings.Split(*members, ",")
		_, err := g.members.Join(parts)
		if err != nil {
			return err
		}
	}

	g.queue = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return g.members.NumMembers()
		},
		RetransmitMult: c.RetransmitMult,
	}

	local := g.members.LocalNode()
	LogPrintf(LOG_DEBUG, "gossip", "local member %s:%d", local.Addr, local.Port)

	g.Ready = true
	return nil
}

// Broadcast message
func (g *Gossip) Broadcast(msg []byte) {
	if !g.Ready || g.queue == nil {
		LogPrintf(LOG_DEBUG, "gossip", "not ready")
		return
	}
	g.queue.QueueBroadcast(&broadcast_impl{
		msg:    msg,
		notify: nil,
		gossip: g,
	})
}

// Generage a default broadcast, need to set handlers:
//   - NotifyMsgHandler
//   - LocalStateHandler
//   - MergeRemoteStateHandler
func DefaultGossip() *Gossip {
	hostname, _ := os.Hostname()
	return GossipWith(hostname+"-"+uuid.New().String(), "", 0)
}

func GossipWith(name string, bindAddr string, bindPort int) *Gossip {
	var g = &Gossip{}
	g.Name = name
	g.BindAddr = bindAddr
	g.BindPort = bindPort
	g.RetransmitMult = 0
	g.Ready = false
	g.NodeMetaHandler = defaultNodeMetaHandler
	g.NotifyJoinHandler = defaultNotifyJoinHandler
	g.NotifyLeaveHandler = defaultNotifyLeaveHandler
	g.NotifyUpdateHandler = defaultNotifyUpdateHandler
	g.InvalidatesHandler = defaultInvalidatesHandler
	return g
}

// Default node meta handler, return nothing
func defaultNodeMetaHandler() []byte {
	return []byte{}
}

// Default node join handler, print join msg
func defaultNotifyJoinHandler(node *memberlist.Node) {
	LogPrintf(LOG_INFO, "gossip", "node '%s' has joined", node.Name)
}

// Default node leave handler, print leave msg
func defaultNotifyLeaveHandler(node *memberlist.Node) {
	LogPrintf(LOG_INFO, "gossip", "node '%s' has left", node.Name)
}

// Default node update handler, print update msg
func defaultNotifyUpdateHandler(node *memberlist.Node) {
	LogPrintf(LOG_INFO, "gossip", "node '%s' has updated", node.Name)
}

// Default msg invalidate handler, return false
func defaultInvalidatesHandler(other memberlist.Broadcast) bool {
	return false
}

// Implatementation of memberlist.Delegate
type delegate_impl struct {
	gossip *Gossip
}

func (d *delegate_impl) NodeMeta(limit int) []byte {
	return d.gossip.NodeMetaHandler()
}

func (d *delegate_impl) NotifyMsg(b []byte) {
	d.gossip.NotifyMsgHandler(b)
}

func (d *delegate_impl) GetBroadcasts(overhead, limit int) [][]byte {
	return d.gossip.queue.GetBroadcasts(overhead, limit)
}

func (d *delegate_impl) LocalState(join bool) []byte {
	return d.gossip.LocalStateHandler()
}

func (d *delegate_impl) MergeRemoteState(buf []byte, join bool) {
	d.gossip.MergeRemoteStateHandler(buf)
}

// Implementation of memberlist.EventDelegate
type event_delegate_impl struct {
	gossip *Gossip
}

func (ed *event_delegate_impl) NotifyJoin(node *memberlist.Node) {
	ed.gossip.NotifyJoinHandler(node)
}

func (ed *event_delegate_impl) NotifyLeave(node *memberlist.Node) {
	ed.gossip.NotifyLeaveHandler(node)
}

func (ed *event_delegate_impl) NotifyUpdate(node *memberlist.Node) {
	ed.gossip.NotifyUpdateHandler(node)
}

// Implementation of memberlist.Broadcast
type broadcast_impl struct {
	msg    []byte
	notify chan<- struct{}
	gossip *Gossip
}

func (b *broadcast_impl) Invalidates(other memberlist.Broadcast) bool {
	return b.gossip.InvalidatesHandler(other)
}

func (b *broadcast_impl) Message() []byte {
	return b.msg
}

func (b *broadcast_impl) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}
