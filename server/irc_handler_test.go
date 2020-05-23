package server

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/khlieng/dispatch/pkg/irc"
	"github.com/khlieng/dispatch/storage"
	"github.com/khlieng/dispatch/storage/bleve"
	"github.com/khlieng/dispatch/storage/boltdb"
)

var user *storage.User

func TestMain(m *testing.M) {
	tempdir, err := ioutil.TempDir("", "test_")
	if err != nil {
		log.Fatal(err)
	}

	storage.Initialize(tempdir, "", "")

	db, err := boltdb.New(storage.Path.Database())
	if err != nil {
		log.Fatal(err)
	}

	user, err = storage.NewUser(db)
	if err != nil {
		log.Fatal(err)
	}
	user.SetMessageStore(db)

	search, err := bleve.New(storage.Path.Index(user.Username))
	if err != nil {
		log.Fatal(err)
	}
	user.SetMessageSearchProvider(search)

	channelStore = storage.NewChannelStore()

	code := m.Run()

	os.RemoveAll(tempdir)
	os.Exit(code)
}

func dispatchMessage(msg *irc.Message) WSResponse {
	return <-dispatchMessageMulti(msg)
}

func dispatchMessageMulti(msg *irc.Message) chan WSResponse {
	c := irc.NewClient("nick", "user")
	c.Host = "host.com"
	s := NewState(user, nil)

	newIRCHandler(c, s).dispatchMessage(msg)

	return s.broadcast
}

func checkResponse(t *testing.T, expectedType string, expectedData interface{}, res WSResponse) {
	assert.Equal(t, expectedType, res.Type)
	assert.Equal(t, expectedData, res.Data)
}

func TestHandleIRCNick(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.NICK,
		Nick:    "old",
		Params:  []string{"new"},
	})

	checkResponse(t, "nick", Nick{
		Server: "host.com",
		Old:    "old",
		New:    "new",
	}, res)
}

func TestHandleIRCJoin(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.JOIN,
		Nick:    "joining",
		Params:  []string{"#chan"},
	})

	checkResponse(t, "join", Join{
		Server:   "host.com",
		User:     "joining",
		Channels: []string{"#chan"},
	}, res)
}

func TestHandleIRCPart(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.PART,
		Nick:    "parting",
		Params:  []string{"#chan", "the reason"},
	})

	checkResponse(t, "part", Part{
		Server:  "host.com",
		User:    "parting",
		Channel: "#chan",
		Reason:  "the reason",
	}, res)

	res = dispatchMessage(&irc.Message{
		Command: irc.PART,
		Nick:    "parting",
		Params:  []string{"#chan"},
	})

	checkResponse(t, "part", Part{
		Server:  "host.com",
		User:    "parting",
		Channel: "#chan",
	}, res)
}

func TestHandleIRCMode(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.MODE,
		Params:  []string{"#chan", "+o-v", "nick"},
	})

	checkResponse(t, "mode", &Mode{
		Server:  "host.com",
		Channel: "#chan",
		User:    "nick",
		Add:     "o",
		Remove:  "v",
	}, res)
}

func TestHandleIRCMessage(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.PRIVMSG,
		Nick:    "nick",
		Params:  []string{"#chan", "the message"},
	})

	assert.Equal(t, "message", res.Type)
	msg, ok := res.Data.(Message)
	assert.True(t, ok)
	assert.Equal(t, "nick", msg.From)
	assert.Equal(t, "#chan", msg.To)
	assert.Equal(t, "the message", msg.Content)

	res = dispatchMessage(&irc.Message{
		Command: irc.PRIVMSG,
		Nick:    "someone",
		Params:  []string{"nick", "the message"},
	})

	assert.Equal(t, "pm", res.Type)
	msg, ok = res.Data.(Message)
	assert.True(t, ok)
	assert.Equal(t, "someone", msg.From)
	assert.Empty(t, msg.To)
	assert.Equal(t, "the message", msg.Content)
}

func TestHandleIRCQuit(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.QUIT,
		Nick:    "nick",
		Params:  []string{"the reason"},
	})

	checkResponse(t, "quit", Quit{
		Server: "host.com",
		User:   "nick",
		Reason: "the reason",
	}, res)
}

func TestHandleIRCWelcome(t *testing.T) {
	res := dispatchMessageMulti(&irc.Message{
		Command: irc.RPL_WELCOME,
		Nick:    "nick",
		Params:  []string{"nick", "some", "text"},
	})

	checkResponse(t, "nick", Nick{
		Server: "host.com",
		New:    "nick",
	}, <-res)

	checkResponse(t, "pm", Message{
		Server:  "host.com",
		From:    "nick",
		Content: "some text",
	}, <-res)
}

func TestHandleIRCWhois(t *testing.T) {
	c := irc.NewClient("nick", "user")
	c.Host = "host.com"
	s := NewState(nil, nil)
	i := newIRCHandler(c, s)

	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_WHOISUSER,
		Params:  []string{"", "nick", "user", "host", "", "realname"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_WHOISSERVER,
		Params:  []string{"", "", "srv.com"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_WHOISCHANNELS,
		Params:  []string{"#chan #chan1"},
	})
	i.dispatchMessage(&irc.Message{Command: irc.RPL_ENDOFWHOIS})

	checkResponse(t, "whois", WhoisReply{
		Nick:     "nick",
		Username: "user",
		Host:     "host",
		Realname: "realname",
		Server:   "srv.com",
		Channels: []string{"#chan", "#chan1"},
	}, <-s.broadcast)
}

func TestHandleIRCTopic(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.RPL_TOPIC,
		Params:  []string{"target", "#chan", "the topic"},
	})

	checkResponse(t, "topic", Topic{
		Server:  "host.com",
		Channel: "#chan",
		Topic:   "the topic",
	}, res)

	res = dispatchMessage(&irc.Message{
		Command: irc.TOPIC,
		Params:  []string{"#chan", "the topic"},
		Nick:    "bob",
	})

	checkResponse(t, "topic", Topic{
		Server:  "host.com",
		Channel: "#chan",
		Topic:   "the topic",
		Nick:    "bob",
	}, res)
}

func TestHandleIRCNoTopic(t *testing.T) {
	res := dispatchMessage(&irc.Message{
		Command: irc.RPL_NOTOPIC,
		Params:  []string{"target", "#chan", "No topic set."},
	})

	checkResponse(t, "topic", Topic{
		Server:  "host.com",
		Channel: "#chan",
	}, res)
}

func TestHandleIRCNames(t *testing.T) {
	c := irc.NewClient("nick", "user")
	c.Host = "host.com"
	s := NewState(nil, nil)
	i := newIRCHandler(c, s)

	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_NAMREPLY,
		Params:  []string{"", "", "#chan", "a b c"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_NAMREPLY,
		Params:  []string{"", "", "#chan", "d"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_ENDOFNAMES,
		Params:  []string{"", "#chan"},
	})

	checkResponse(t, "users", Userlist{
		Server:  "host.com",
		Channel: "#chan",
		Users:   []string{"a", "b", "c", "d"},
	}, <-s.broadcast)
}

func TestHandleIRCMotd(t *testing.T) {
	c := irc.NewClient("nick", "user")
	c.Host = "host.com"
	s := NewState(nil, nil)
	i := newIRCHandler(c, s)

	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_MOTDSTART,
		Params:  []string{"motd title"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_MOTD,
		Params:  []string{"line 1"},
	})
	i.dispatchMessage(&irc.Message{
		Command: irc.RPL_MOTD,
		Params:  []string{"line 2"},
	})
	i.dispatchMessage(&irc.Message{Command: irc.RPL_ENDOFMOTD})

	checkResponse(t, "motd", MOTD{
		Server:  "host.com",
		Title:   "motd title",
		Content: []string{"line 1", "line 2"},
	}, <-s.broadcast)
}

func TestHandleIRCBadNick(t *testing.T) {
	c := irc.NewClient("nick", "user")
	c.Host = "host.com"
	s := NewState(nil, nil)
	i := newIRCHandler(c, s)

	i.dispatchMessage(&irc.Message{
		Command: irc.ERR_ERRONEUSNICKNAME,
	})

	// It should print the error message first
	<-s.broadcast

	checkResponse(t, "nick_fail", NickFail{
		Server: "host.com",
	}, <-s.broadcast)
}
