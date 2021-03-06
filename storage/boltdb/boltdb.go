package boltdb

import (
	"bytes"
	"encoding/binary"
	"strconv"

	bolt "go.etcd.io/bbolt"

	"github.com/khlieng/dispatch/pkg/session"
	"github.com/khlieng/dispatch/storage"
)

var (
	bucketUsers    = []byte("Users")
	bucketNetworks = []byte("Networks")
	bucketChannels = []byte("Channels")
	bucketOpenDMs  = []byte("OpenDMs")
	bucketMessages = []byte("Messages")
	bucketSessions = []byte("Sessions")
)

// BoltStore implements storage.Store, storage.MessageStore and storage.SessionStore
type BoltStore struct {
	db *bolt.DB
}

func New(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(bucketUsers)
		tx.CreateBucketIfNotExists(bucketNetworks)
		tx.CreateBucketIfNotExists(bucketChannels)
		tx.CreateBucketIfNotExists(bucketOpenDMs)
		tx.CreateBucketIfNotExists(bucketMessages)
		tx.CreateBucketIfNotExists(bucketSessions)
		return nil
	})

	return &BoltStore{
		db,
	}, nil
}

func (s *BoltStore) Close() {
	s.db.Close()
}

func (s *BoltStore) Users() ([]*storage.User, error) {
	var users []*storage.User

	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)

		return b.ForEach(func(k, v []byte) error {
			user := storage.User{
				IDBytes: make([]byte, 8),
			}
			user.Unmarshal(v)
			copy(user.IDBytes, k)

			users = append(users, &user)

			return nil
		})
	})

	return users, nil
}

func (s *BoltStore) SaveUser(user *storage.User) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUsers)

		if user.ID == 0 {
			user.ID, _ = b.NextSequence()
			user.IDBytes = idToBytes(user.ID)
		}
		user.Username = strconv.FormatUint(user.ID, 10)

		data, err := user.Marshal(nil)
		if err != nil {
			return err
		}

		return b.Put(user.IDBytes, data)
	})
}

func (s *BoltStore) DeleteUser(user *storage.User) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		err := tx.Bucket(bucketUsers).Delete(user.IDBytes)
		if err != nil {
			return err
		}

		return deletePrefix(user.IDBytes,
			tx.Bucket(bucketNetworks),
			tx.Bucket(bucketChannels),
			tx.Bucket(bucketOpenDMs),
		)
	})
}

func (s *BoltStore) Network(user *storage.User, address string) (*storage.Network, error) {
	var network *storage.Network

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		id := networkID(user, address)

		v := b.Get(id)
		if v == nil {
			return storage.ErrNotFound
		} else {
			network = &storage.Network{}
			network.Unmarshal(v)
			return nil
		}
	})

	return network, err
}

func (s *BoltStore) Networks(user *storage.User) ([]*storage.Network, error) {
	var networks []*storage.Network

	s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketNetworks).Cursor()

		for k, v := c.Seek(user.IDBytes); bytes.HasPrefix(k, user.IDBytes); k, v = c.Next() {
			network := storage.Network{}
			network.Unmarshal(v)
			networks = append(networks, &network)
		}

		return nil
	})

	return networks, nil
}

func (s *BoltStore) SaveNetwork(user *storage.User, network *storage.Network) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		data, _ := network.Marshal(nil)

		return b.Put(networkID(user, network.Host), data)
	})
}

func (s *BoltStore) RemoveNetwork(user *storage.User, address string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		networkID := networkID(user, address)
		err := tx.Bucket(bucketNetworks).Delete(networkID)
		if err != nil {
			return err
		}

		return deletePrefix(networkID,
			tx.Bucket(bucketChannels),
			tx.Bucket(bucketOpenDMs),
		)
	})
}

func (s *BoltStore) SetNick(user *storage.User, nick, address string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		id := networkID(user, address)

		network := storage.Network{}
		v := b.Get(id)
		if v != nil {
			network.Unmarshal(v)
			network.Nick = nick

			data, _ := network.Marshal(nil)
			return b.Put(id, data)
		}

		return nil
	})
}

func (s *BoltStore) SetNetworkName(user *storage.User, name, address string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		id := networkID(user, address)

		network := storage.Network{}
		v := b.Get(id)
		if v != nil {
			network.Unmarshal(v)
			network.Name = name

			data, _ := network.Marshal(nil)
			return b.Put(id, data)
		}

		return nil
	})
}

func (s *BoltStore) Channels(user *storage.User) ([]*storage.Channel, error) {
	var channels []*storage.Channel

	s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketChannels).Cursor()

		for k, v := c.Seek(user.IDBytes); bytes.HasPrefix(k, user.IDBytes); k, v = c.Next() {
			channel := storage.Channel{}
			channel.Unmarshal(v)
			channels = append(channels, &channel)
		}

		return nil
	})

	return channels, nil
}

func (s *BoltStore) SaveChannel(user *storage.User, channel *storage.Channel) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketChannels)
		data, _ := channel.Marshal(nil)

		return b.Put(channelID(user, channel.Network, channel.Name), data)
	})
}

func (s *BoltStore) RemoveChannel(user *storage.User, network, channel string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketChannels)
		id := channelID(user, network, channel)

		return b.Delete(id)
	})
}

func (s *BoltStore) HasChannel(user *storage.User, network, channel string) bool {
	has := false
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketChannels)
		has = b.Get(channelID(user, network, channel)) != nil

		return nil
	})
	return has
}

func (s *BoltStore) OpenDMs(user *storage.User) ([]storage.Tab, error) {
	var openDMs []storage.Tab

	s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketOpenDMs).Cursor()

		for k, _ := c.Seek(user.IDBytes); bytes.HasPrefix(k, user.IDBytes); k, _ = c.Next() {
			tab := bytes.Split(k[8:], []byte{0})
			openDMs = append(openDMs, storage.Tab{
				Network: string(tab[0]),
				Name:    string(tab[1]),
			})
		}

		return nil
	})

	return openDMs, nil
}

func (s *BoltStore) AddOpenDM(user *storage.User, network, nick string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOpenDMs)

		return b.Put(channelID(user, network, nick), nil)
	})
}

func (s *BoltStore) RemoveOpenDM(user *storage.User, network, nick string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOpenDMs)

		return b.Delete(channelID(user, network, nick))
	})
}

func (s *BoltStore) logMessage(tx *bolt.Tx, message *storage.Message) error {
	b, err := tx.Bucket(bucketMessages).CreateBucketIfNotExists([]byte(message.Network + ":" + message.To))
	if err != nil {
		return err
	}

	data, err := message.Marshal(nil)
	if err != nil {
		return err
	}

	return b.Put([]byte(message.ID), data)
}

func (s *BoltStore) LogMessage(message *storage.Message) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		return s.logMessage(tx, message)
	})
}

func (s *BoltStore) LogMessages(messages []*storage.Message) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		for _, message := range messages {
			err := s.logMessage(tx, message)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *BoltStore) Messages(network, channel string, count int, fromID string) ([]storage.Message, bool, error) {
	messages := make([]storage.Message, count)
	hasMore := false

	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMessages).Bucket([]byte(network + ":" + channel))
		if b == nil {
			return nil
		}

		c := b.Cursor()

		if fromID != "" {
			c.Seek([]byte(fromID))

			for k, v := c.Prev(); count > 0 && k != nil; k, v = c.Prev() {
				count--
				messages[count].Unmarshal(v)
			}
		} else {
			for k, v := c.Last(); count > 0 && k != nil; k, v = c.Prev() {
				count--
				messages[count].Unmarshal(v)
			}
		}

		c.Next()
		k, _ := c.Prev()
		hasMore = k != nil

		return nil
	})

	if count == 0 {
		return messages, hasMore, nil
	} else if count < len(messages) {
		return messages[count:], hasMore, nil
	}

	return nil, false, nil
}

func (s *BoltStore) MessagesByID(network, channel string, ids []string) ([]storage.Message, error) {
	messages := make([]storage.Message, len(ids))

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMessages).Bucket([]byte(network + ":" + channel))

		for i, id := range ids {
			messages[i].Unmarshal(b.Get([]byte(id)))
		}
		return nil
	})
	return messages, err
}

func (s *BoltStore) Sessions() ([]*session.Session, error) {
	var sessions []*session.Session

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSessions)

		return b.ForEach(func(_ []byte, v []byte) error {
			session := session.Session{}
			_, err := session.Unmarshal(v)
			sessions = append(sessions, &session)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *BoltStore) SaveSession(session *session.Session) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSessions)

		data, err := session.Marshal(nil)
		if err != nil {
			return err
		}

		return b.Put([]byte(session.Key()), data)
	})
}

func (s *BoltStore) DeleteSession(key string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSessions).Delete([]byte(key))
	})
}

func deletePrefix(prefix []byte, buckets ...*bolt.Bucket) error {
	for _, b := range buckets {
		c := b.Cursor()

		for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			err := b.Delete(k)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func networkID(user *storage.User, address string) []byte {
	id := make([]byte, 8+len(address))
	copy(id, user.IDBytes)
	copy(id[8:], address)
	return id
}

func channelID(user *storage.User, network, channel string) []byte {
	id := make([]byte, 8+len(network)+1+len(channel))
	copy(id, user.IDBytes)
	copy(id[8:], network)
	copy(id[8+len(network)+1:], channel)
	return id
}

func idToBytes(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

func idFromBytes(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
