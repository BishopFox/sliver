// Discordgo - Discord bindings for Go
// Available at https://github.com/bwmarrin/discordgo

// Copyright 2015-2016 Bruce Marriner <bruce@sqls.net>.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains code related to state tracking.  If enabled, state
// tracking will capture the initial READY packet and many other websocket
// events and maintain an in-memory state of guilds, channels, users, and
// so forth.  This information can be accessed through the Session.State struct.

package discordgo

import (
	"errors"
	"sort"
	"sync"
)

// ErrNilState is returned when the state is nil.
var ErrNilState = errors.New("state not instantiated, please use discordgo.New() or assign Session.State")

// ErrStateNotFound is returned when the state cache
// requested is not found
var ErrStateNotFound = errors.New("state cache not found")

// ErrMessageIncompletePermissions is returned when the message
// requested for permissions does not contain enough data to
// generate the permissions.
var ErrMessageIncompletePermissions = errors.New("message incomplete, unable to determine permissions")

// A State contains the current known state.
// As discord sends this in a READY blob, it seems reasonable to simply
// use that struct as the data store.
type State struct {
	sync.RWMutex
	Ready

	// MaxMessageCount represents how many messages per channel the state will store.
	MaxMessageCount    int
	TrackChannels      bool
	TrackThreads       bool
	TrackEmojis        bool
	TrackStickers      bool
	TrackMembers       bool
	TrackThreadMembers bool
	TrackRoles         bool
	TrackVoice         bool
	TrackPresences     bool

	guildMap   map[string]*Guild
	channelMap map[string]*Channel
	memberMap  map[string]map[string]*Member
}

// NewState creates an empty state.
func NewState() *State {
	return &State{
		Ready: Ready{
			PrivateChannels: []*Channel{},
			Guilds:          []*Guild{},
		},
		TrackChannels:      true,
		TrackThreads:       true,
		TrackEmojis:        true,
		TrackStickers:      true,
		TrackMembers:       true,
		TrackThreadMembers: true,
		TrackRoles:         true,
		TrackVoice:         true,
		TrackPresences:     true,
		guildMap:           make(map[string]*Guild),
		channelMap:         make(map[string]*Channel),
		memberMap:          make(map[string]map[string]*Member),
	}
}

func (s *State) createMemberMap(guild *Guild) {
	members := make(map[string]*Member)
	for _, m := range guild.Members {
		members[m.User.ID] = m
	}
	s.memberMap[guild.ID] = members
}

// GuildAdd adds a guild to the current world state, or
// updates it if it already exists.
func (s *State) GuildAdd(guild *Guild) error {
	if s == nil {
		return ErrNilState
	}

	s.Lock()
	defer s.Unlock()

	// Update the channels to point to the right guild, adding them to the channelMap as we go
	for _, c := range guild.Channels {
		s.channelMap[c.ID] = c
	}

	// Add all the threads to the state in case of thread sync list.
	for _, t := range guild.Threads {
		s.channelMap[t.ID] = t
	}

	// If this guild contains a new member slice, we must regenerate the member map so the pointers stay valid
	if guild.Members != nil {
		s.createMemberMap(guild)
	} else if _, ok := s.memberMap[guild.ID]; !ok {
		// Even if we have no new member slice, we still initialize the member map for this guild if it doesn't exist
		s.memberMap[guild.ID] = make(map[string]*Member)
	}

	if g, ok := s.guildMap[guild.ID]; ok {
		// We are about to replace `g` in the state with `guild`, but first we need to
		// make sure we preserve any fields that the `guild` doesn't contain from `g`.
		if guild.MemberCount == 0 {
			guild.MemberCount = g.MemberCount
		}
		if guild.Roles == nil {
			guild.Roles = g.Roles
		}
		if guild.Emojis == nil {
			guild.Emojis = g.Emojis
		}
		if guild.Members == nil {
			guild.Members = g.Members
		}
		if guild.Presences == nil {
			guild.Presences = g.Presences
		}
		if guild.Channels == nil {
			guild.Channels = g.Channels
		}
		if guild.Threads == nil {
			guild.Threads = g.Threads
		}
		if guild.VoiceStates == nil {
			guild.VoiceStates = g.VoiceStates
		}
		*g = *guild
		return nil
	}

	s.Guilds = append(s.Guilds, guild)
	s.guildMap[guild.ID] = guild

	return nil
}

// GuildRemove removes a guild from current world state.
func (s *State) GuildRemove(guild *Guild) error {
	if s == nil {
		return ErrNilState
	}

	_, err := s.Guild(guild.ID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	delete(s.guildMap, guild.ID)

	for i, g := range s.Guilds {
		if g.ID == guild.ID {
			s.Guilds = append(s.Guilds[:i], s.Guilds[i+1:]...)
			return nil
		}
	}

	return nil
}

// Guild gets a guild by ID.
// Useful for querying if @me is in a guild:
//    _, err := discordgo.Session.State.Guild(guildID)
//	  isInGuild := err == nil
func (s *State) Guild(guildID string) (*Guild, error) {
	if s == nil {
		return nil, ErrNilState
	}

	s.RLock()
	defer s.RUnlock()

	if g, ok := s.guildMap[guildID]; ok {
		return g, nil
	}

	return nil, ErrStateNotFound
}

func (s *State) presenceAdd(guildID string, presence *Presence) error {
	guild, ok := s.guildMap[guildID]
	if !ok {
		return ErrStateNotFound
	}

	for i, p := range guild.Presences {
		if p.User.ID == presence.User.ID {
			//guild.Presences[i] = presence

			//Update status
			guild.Presences[i].Activities = presence.Activities
			if presence.Status != "" {
				guild.Presences[i].Status = presence.Status
			}
			if presence.ClientStatus.Desktop != "" {
				guild.Presences[i].ClientStatus.Desktop = presence.ClientStatus.Desktop
			}
			if presence.ClientStatus.Mobile != "" {
				guild.Presences[i].ClientStatus.Mobile = presence.ClientStatus.Mobile
			}
			if presence.ClientStatus.Web != "" {
				guild.Presences[i].ClientStatus.Web = presence.ClientStatus.Web
			}

			//Update the optionally sent user information
			//ID Is a mandatory field so you should not need to check if it is empty
			guild.Presences[i].User.ID = presence.User.ID

			if presence.User.Avatar != "" {
				guild.Presences[i].User.Avatar = presence.User.Avatar
			}
			if presence.User.Discriminator != "" {
				guild.Presences[i].User.Discriminator = presence.User.Discriminator
			}
			if presence.User.Email != "" {
				guild.Presences[i].User.Email = presence.User.Email
			}
			if presence.User.Token != "" {
				guild.Presences[i].User.Token = presence.User.Token
			}
			if presence.User.Username != "" {
				guild.Presences[i].User.Username = presence.User.Username
			}

			return nil
		}
	}

	guild.Presences = append(guild.Presences, presence)
	return nil
}

// PresenceAdd adds a presence to the current world state, or
// updates it if it already exists.
func (s *State) PresenceAdd(guildID string, presence *Presence) error {
	if s == nil {
		return ErrNilState
	}

	s.Lock()
	defer s.Unlock()

	return s.presenceAdd(guildID, presence)
}

// PresenceRemove removes a presence from the current world state.
func (s *State) PresenceRemove(guildID string, presence *Presence) error {
	if s == nil {
		return ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for i, p := range guild.Presences {
		if p.User.ID == presence.User.ID {
			guild.Presences = append(guild.Presences[:i], guild.Presences[i+1:]...)
			return nil
		}
	}

	return ErrStateNotFound
}

// Presence gets a presence by ID from a guild.
func (s *State) Presence(guildID, userID string) (*Presence, error) {
	if s == nil {
		return nil, ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return nil, err
	}

	for _, p := range guild.Presences {
		if p.User.ID == userID {
			return p, nil
		}
	}

	return nil, ErrStateNotFound
}

// TODO: Consider moving Guild state update methods onto *Guild.

func (s *State) memberAdd(member *Member) error {
	guild, ok := s.guildMap[member.GuildID]
	if !ok {
		return ErrStateNotFound
	}

	members, ok := s.memberMap[member.GuildID]
	if !ok {
		return ErrStateNotFound
	}

	m, ok := members[member.User.ID]
	if !ok {
		members[member.User.ID] = member
		guild.Members = append(guild.Members, member)
	} else {
		// We are about to replace `m` in the state with `member`, but first we need to
		// make sure we preserve any fields that the `member` doesn't contain from `m`.
		if member.JoinedAt.IsZero() {
			member.JoinedAt = m.JoinedAt
		}
		*m = *member
	}
	return nil
}

// MemberAdd adds a member to the current world state, or
// updates it if it already exists.
func (s *State) MemberAdd(member *Member) error {
	if s == nil {
		return ErrNilState
	}

	s.Lock()
	defer s.Unlock()

	return s.memberAdd(member)
}

// MemberRemove removes a member from current world state.
func (s *State) MemberRemove(member *Member) error {
	if s == nil {
		return ErrNilState
	}

	guild, err := s.Guild(member.GuildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	members, ok := s.memberMap[member.GuildID]
	if !ok {
		return ErrStateNotFound
	}

	_, ok = members[member.User.ID]
	if !ok {
		return ErrStateNotFound
	}
	delete(members, member.User.ID)

	for i, m := range guild.Members {
		if m.User.ID == member.User.ID {
			guild.Members = append(guild.Members[:i], guild.Members[i+1:]...)
			return nil
		}
	}

	return ErrStateNotFound
}

// Member gets a member by ID from a guild.
func (s *State) Member(guildID, userID string) (*Member, error) {
	if s == nil {
		return nil, ErrNilState
	}

	s.RLock()
	defer s.RUnlock()

	members, ok := s.memberMap[guildID]
	if !ok {
		return nil, ErrStateNotFound
	}

	m, ok := members[userID]
	if ok {
		return m, nil
	}

	return nil, ErrStateNotFound
}

// RoleAdd adds a role to the current world state, or
// updates it if it already exists.
func (s *State) RoleAdd(guildID string, role *Role) error {
	if s == nil {
		return ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for i, r := range guild.Roles {
		if r.ID == role.ID {
			guild.Roles[i] = role
			return nil
		}
	}

	guild.Roles = append(guild.Roles, role)
	return nil
}

// RoleRemove removes a role from current world state by ID.
func (s *State) RoleRemove(guildID, roleID string) error {
	if s == nil {
		return ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for i, r := range guild.Roles {
		if r.ID == roleID {
			guild.Roles = append(guild.Roles[:i], guild.Roles[i+1:]...)
			return nil
		}
	}

	return ErrStateNotFound
}

// Role gets a role by ID from a guild.
func (s *State) Role(guildID, roleID string) (*Role, error) {
	if s == nil {
		return nil, ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	for _, r := range guild.Roles {
		if r.ID == roleID {
			return r, nil
		}
	}

	return nil, ErrStateNotFound
}

// ChannelAdd adds a channel to the current world state, or
// updates it if it already exists.
// Channels may exist either as PrivateChannels or inside
// a guild.
func (s *State) ChannelAdd(channel *Channel) error {
	if s == nil {
		return ErrNilState
	}

	s.Lock()
	defer s.Unlock()

	// If the channel exists, replace it
	if c, ok := s.channelMap[channel.ID]; ok {
		if channel.Messages == nil {
			channel.Messages = c.Messages
		}
		if channel.PermissionOverwrites == nil {
			channel.PermissionOverwrites = c.PermissionOverwrites
		}
		if channel.ThreadMetadata == nil {
			channel.ThreadMetadata = c.ThreadMetadata
		}

		*c = *channel
		return nil
	}

	if channel.Type == ChannelTypeDM || channel.Type == ChannelTypeGroupDM {
		s.PrivateChannels = append(s.PrivateChannels, channel)
		s.channelMap[channel.ID] = channel
		return nil
	}

	guild, ok := s.guildMap[channel.GuildID]
	if !ok {
		return ErrStateNotFound
	}

	if channel.IsThread() {
		guild.Threads = append(guild.Threads, channel)
	} else {
		guild.Channels = append(guild.Channels, channel)
	}

	s.channelMap[channel.ID] = channel

	return nil
}

// ChannelRemove removes a channel from current world state.
func (s *State) ChannelRemove(channel *Channel) error {
	if s == nil {
		return ErrNilState
	}

	_, err := s.Channel(channel.ID)
	if err != nil {
		return err
	}

	if channel.Type == ChannelTypeDM || channel.Type == ChannelTypeGroupDM {
		s.Lock()
		defer s.Unlock()

		for i, c := range s.PrivateChannels {
			if c.ID == channel.ID {
				s.PrivateChannels = append(s.PrivateChannels[:i], s.PrivateChannels[i+1:]...)
				break
			}
		}
		delete(s.channelMap, channel.ID)
		return nil
	}

	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	if channel.IsThread() {
		for i, t := range guild.Threads {
			if t.ID == channel.ID {
				guild.Threads = append(guild.Threads[:i], guild.Threads[i+1:]...)
				break
			}
		}
	} else {
		for i, c := range guild.Channels {
			if c.ID == channel.ID {
				guild.Channels = append(guild.Channels[:i], guild.Channels[i+1:]...)
				break
			}
		}
	}

	delete(s.channelMap, channel.ID)

	return nil
}

// ThreadListSync syncs guild threads with provided ones.
func (s *State) ThreadListSync(tls *ThreadListSync) error {
	guild, err := s.Guild(tls.GuildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	// This algorithm filters out archived or
	// threads which are children of channels in channelIDs
	// and then it adds all synced threads to guild threads and cache
	index := 0
outer:
	for _, t := range guild.Threads {
		if !t.ThreadMetadata.Archived && tls.ChannelIDs != nil {
			for _, v := range tls.ChannelIDs {
				if t.ParentID == v {
					delete(s.channelMap, t.ID)
					continue outer
				}
			}
			guild.Threads[index] = t
			index++
		} else {
			delete(s.channelMap, t.ID)
		}
	}
	guild.Threads = guild.Threads[:index]
	for _, t := range tls.Threads {
		s.channelMap[t.ID] = t
		guild.Threads = append(guild.Threads, t)
	}

	for _, m := range tls.Members {
		if c, ok := s.channelMap[m.ID]; ok {
			c.Member = m
		}
	}

	return nil
}

// ThreadMembersUpdate updates thread members list
func (s *State) ThreadMembersUpdate(tmu *ThreadMembersUpdate) error {
	thread, err := s.Channel(tmu.ID)
	if err != nil {
		return err
	}
	s.Lock()
	defer s.Unlock()

	for idx, member := range thread.Members {
		for _, removedMember := range tmu.RemovedMembers {
			if member.ID == removedMember {
				thread.Members = append(thread.Members[:idx], thread.Members[idx+1:]...)
				break
			}
		}
	}

	for _, addedMember := range tmu.AddedMembers {
		thread.Members = append(thread.Members, addedMember.ThreadMember)
		if addedMember.Member != nil {
			err = s.memberAdd(addedMember.Member)
			if err != nil {
				return err
			}
		}
		if addedMember.Presence != nil {
			err = s.presenceAdd(tmu.GuildID, addedMember.Presence)
			if err != nil {
				return err
			}
		}
	}
	thread.MemberCount = tmu.MemberCount

	return nil
}

// ThreadMemberUpdate sets or updates member data for the current user.
func (s *State) ThreadMemberUpdate(mu *ThreadMemberUpdate) error {
	thread, err := s.Channel(mu.ID)
	if err != nil {
		return err
	}

	thread.Member = mu.ThreadMember
	return nil
}

// Channel gets a channel by ID, it will look in all guilds and private channels.
func (s *State) Channel(channelID string) (*Channel, error) {
	if s == nil {
		return nil, ErrNilState
	}

	s.RLock()
	defer s.RUnlock()

	if c, ok := s.channelMap[channelID]; ok {
		return c, nil
	}

	return nil, ErrStateNotFound
}

// Emoji returns an emoji for a guild and emoji id.
func (s *State) Emoji(guildID, emojiID string) (*Emoji, error) {
	if s == nil {
		return nil, ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	for _, e := range guild.Emojis {
		if e.ID == emojiID {
			return e, nil
		}
	}

	return nil, ErrStateNotFound
}

// EmojiAdd adds an emoji to the current world state.
func (s *State) EmojiAdd(guildID string, emoji *Emoji) error {
	if s == nil {
		return ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for i, e := range guild.Emojis {
		if e.ID == emoji.ID {
			guild.Emojis[i] = emoji
			return nil
		}
	}

	guild.Emojis = append(guild.Emojis, emoji)
	return nil
}

// EmojisAdd adds multiple emojis to the world state.
func (s *State) EmojisAdd(guildID string, emojis []*Emoji) error {
	for _, e := range emojis {
		if err := s.EmojiAdd(guildID, e); err != nil {
			return err
		}
	}
	return nil
}

// MessageAdd adds a message to the current world state, or updates it if it exists.
// If the channel cannot be found, the message is discarded.
// Messages are kept in state up to s.MaxMessageCount per channel.
func (s *State) MessageAdd(message *Message) error {
	if s == nil {
		return ErrNilState
	}

	c, err := s.Channel(message.ChannelID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	// If the message exists, merge in the new message contents.
	for _, m := range c.Messages {
		if m.ID == message.ID {
			if message.Content != "" {
				m.Content = message.Content
			}
			if message.EditedTimestamp != nil {
				m.EditedTimestamp = message.EditedTimestamp
			}
			if message.Mentions != nil {
				m.Mentions = message.Mentions
			}
			if message.Embeds != nil {
				m.Embeds = message.Embeds
			}
			if message.Attachments != nil {
				m.Attachments = message.Attachments
			}
			if !message.Timestamp.IsZero() {
				m.Timestamp = message.Timestamp
			}
			if message.Author != nil {
				m.Author = message.Author
			}
			if message.Components != nil {
				m.Components = message.Components
			}

			return nil
		}
	}

	c.Messages = append(c.Messages, message)

	if len(c.Messages) > s.MaxMessageCount {
		c.Messages = c.Messages[len(c.Messages)-s.MaxMessageCount:]
	}

	return nil
}

// MessageRemove removes a message from the world state.
func (s *State) MessageRemove(message *Message) error {
	if s == nil {
		return ErrNilState
	}

	return s.messageRemoveByID(message.ChannelID, message.ID)
}

// messageRemoveByID removes a message by channelID and messageID from the world state.
func (s *State) messageRemoveByID(channelID, messageID string) error {
	c, err := s.Channel(channelID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for i, m := range c.Messages {
		if m.ID == messageID {
			c.Messages = append(c.Messages[:i], c.Messages[i+1:]...)

			return nil
		}
	}

	return ErrStateNotFound
}

func (s *State) voiceStateUpdate(update *VoiceStateUpdate) error {
	guild, err := s.Guild(update.GuildID)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	// Handle Leaving Channel
	if update.ChannelID == "" {
		for i, state := range guild.VoiceStates {
			if state.UserID == update.UserID {
				guild.VoiceStates = append(guild.VoiceStates[:i], guild.VoiceStates[i+1:]...)
				return nil
			}
		}
	} else {
		for i, state := range guild.VoiceStates {
			if state.UserID == update.UserID {
				guild.VoiceStates[i] = update.VoiceState
				return nil
			}
		}

		guild.VoiceStates = append(guild.VoiceStates, update.VoiceState)
	}

	return nil
}

// VoiceState gets a VoiceState by guild and user ID.
func (s *State) VoiceState(guildID, userID string) (*VoiceState, error) {
	if s == nil {
		return nil, ErrNilState
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return nil, err
	}

	for _, state := range guild.VoiceStates {
		if state.UserID == userID {
			return state, nil
		}
	}

	return nil, ErrStateNotFound
}

// Message gets a message by channel and message ID.
func (s *State) Message(channelID, messageID string) (*Message, error) {
	if s == nil {
		return nil, ErrNilState
	}

	c, err := s.Channel(channelID)
	if err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	for _, m := range c.Messages {
		if m.ID == messageID {
			return m, nil
		}
	}

	return nil, ErrStateNotFound
}

// OnReady takes a Ready event and updates all internal state.
func (s *State) onReady(se *Session, r *Ready) (err error) {
	if s == nil {
		return ErrNilState
	}

	s.Lock()
	defer s.Unlock()

	// We must track at least the current user for Voice, even
	// if state is disabled, store the bare essentials.
	if !se.StateEnabled {
		ready := Ready{
			Version:     r.Version,
			SessionID:   r.SessionID,
			User:        r.User,
			Shard:       r.Shard,
			Application: r.Application,
		}

		s.Ready = ready

		return nil
	}

	s.Ready = *r

	for _, g := range s.Guilds {
		s.guildMap[g.ID] = g
		s.createMemberMap(g)

		for _, c := range g.Channels {
			s.channelMap[c.ID] = c
		}
	}

	for _, c := range s.PrivateChannels {
		s.channelMap[c.ID] = c
	}

	return nil
}

// OnInterface handles all events related to states.
func (s *State) OnInterface(se *Session, i interface{}) (err error) {
	if s == nil {
		return ErrNilState
	}

	r, ok := i.(*Ready)
	if ok {
		return s.onReady(se, r)
	}

	if !se.StateEnabled {
		return nil
	}

	switch t := i.(type) {
	case *GuildCreate:
		err = s.GuildAdd(t.Guild)
	case *GuildUpdate:
		err = s.GuildAdd(t.Guild)
	case *GuildDelete:
		var old *Guild
		old, err = s.Guild(t.ID)
		if err == nil {
			oldCopy := *old
			t.BeforeDelete = &oldCopy
		}

		err = s.GuildRemove(t.Guild)
	case *GuildMemberAdd:
		var guild *Guild
		// Updates the MemberCount of the guild.
		guild, err = s.Guild(t.Member.GuildID)
		if err != nil {
			return err
		}
		guild.MemberCount++

		// Caches member if tracking is enabled.
		if s.TrackMembers {
			err = s.MemberAdd(t.Member)
		}
	case *GuildMemberUpdate:
		if s.TrackMembers {
			var old *Member
			old, err = s.Member(t.GuildID, t.User.ID)
			if err == nil {
				oldCopy := *old
				t.BeforeUpdate = &oldCopy
			}

			err = s.MemberAdd(t.Member)
		}
	case *GuildMemberRemove:
		var guild *Guild
		// Updates the MemberCount of the guild.
		guild, err = s.Guild(t.Member.GuildID)
		if err != nil {
			return err
		}
		guild.MemberCount--

		// Removes member from the cache if tracking is enabled.
		if s.TrackMembers {
			err = s.MemberRemove(t.Member)
		}
	case *GuildMembersChunk:
		if s.TrackMembers {
			for i := range t.Members {
				t.Members[i].GuildID = t.GuildID
				err = s.MemberAdd(t.Members[i])
			}
		}

		if s.TrackPresences {
			for _, p := range t.Presences {
				err = s.PresenceAdd(t.GuildID, p)
			}
		}
	case *GuildRoleCreate:
		if s.TrackRoles {
			err = s.RoleAdd(t.GuildID, t.Role)
		}
	case *GuildRoleUpdate:
		if s.TrackRoles {
			err = s.RoleAdd(t.GuildID, t.Role)
		}
	case *GuildRoleDelete:
		if s.TrackRoles {
			err = s.RoleRemove(t.GuildID, t.RoleID)
		}
	case *GuildEmojisUpdate:
		if s.TrackEmojis {
			var guild *Guild
			guild, err = s.Guild(t.GuildID)
			if err != nil {
				return err
			}
			s.Lock()
			defer s.Unlock()
			guild.Emojis = t.Emojis
		}
	case *GuildStickersUpdate:
		if s.TrackStickers {
			var guild *Guild
			guild, err = s.Guild(t.GuildID)
			if err != nil {
				return err
			}
			s.Lock()
			defer s.Unlock()
			guild.Stickers = t.Stickers
		}
	case *ChannelCreate:
		if s.TrackChannels {
			err = s.ChannelAdd(t.Channel)
		}
	case *ChannelUpdate:
		if s.TrackChannels {
			old, err := s.Channel(t.ID)
			if err == nil {
				oldCopy := *old
				t.BeforeUpdate = &oldCopy
			}
			err = s.ChannelAdd(t.Channel)
		}
	case *ChannelDelete:
		if s.TrackChannels {
			err = s.ChannelRemove(t.Channel)
		}
	case *ThreadCreate:
		if s.TrackThreads {
			err = s.ChannelAdd(t.Channel)
		}
	case *ThreadUpdate:
		if s.TrackThreads {
			old, err := s.Channel(t.ID)
			if err == nil {
				oldCopy := *old
				t.BeforeUpdate = &oldCopy
			}
			err = s.ChannelAdd(t.Channel)
		}
	case *ThreadDelete:
		if s.TrackThreads {
			err = s.ChannelRemove(t.Channel)
		}
	case *ThreadMemberUpdate:
		if s.TrackThreads {
			err = s.ThreadMemberUpdate(t)
		}
	case *ThreadMembersUpdate:
		if s.TrackThreadMembers {
			err = s.ThreadMembersUpdate(t)
		}
	case *ThreadListSync:
		if s.TrackThreads {
			err = s.ThreadListSync(t)
		}
	case *MessageCreate:
		if s.MaxMessageCount != 0 {
			err = s.MessageAdd(t.Message)
		}
	case *MessageUpdate:
		if s.MaxMessageCount != 0 {
			var old *Message
			old, err = s.Message(t.ChannelID, t.ID)
			if err == nil {
				oldCopy := *old
				t.BeforeUpdate = &oldCopy
			}

			err = s.MessageAdd(t.Message)
		}
	case *MessageDelete:
		if s.MaxMessageCount != 0 {
			var old *Message
			old, err = s.Message(t.ChannelID, t.ID)
			if err == nil {
				oldCopy := *old
				t.BeforeDelete = &oldCopy
			}

			err = s.MessageRemove(t.Message)
		}
	case *MessageDeleteBulk:
		if s.MaxMessageCount != 0 {
			for _, mID := range t.Messages {
				s.messageRemoveByID(t.ChannelID, mID)
			}
		}
	case *VoiceStateUpdate:
		if s.TrackVoice {
			var old *VoiceState
			old, err = s.VoiceState(t.GuildID, t.UserID)
			if err == nil {
				oldCopy := *old
				t.BeforeUpdate = &oldCopy
			}

			err = s.voiceStateUpdate(t)
		}
	case *PresenceUpdate:
		if s.TrackPresences {
			s.PresenceAdd(t.GuildID, &t.Presence)
		}
		if s.TrackMembers {
			if t.Status == StatusOffline {
				return
			}

			var m *Member
			m, err = s.Member(t.GuildID, t.User.ID)

			if err != nil {
				// Member not found; this is a user coming online
				m = &Member{
					GuildID: t.GuildID,
					User:    t.User,
				}
			} else {
				if t.User.Username != "" {
					m.User.Username = t.User.Username
				}
			}

			err = s.MemberAdd(m)
		}

	}

	return
}

// UserChannelPermissions returns the permission of a user in a channel.
// userID    : The ID of the user to calculate permissions for.
// channelID : The ID of the channel to calculate permission for.
func (s *State) UserChannelPermissions(userID, channelID string) (apermissions int64, err error) {
	if s == nil {
		return 0, ErrNilState
	}

	channel, err := s.Channel(channelID)
	if err != nil {
		return
	}

	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		return
	}

	member, err := s.Member(guild.ID, userID)
	if err != nil {
		return
	}

	return memberPermissions(guild, channel, userID, member.Roles), nil
}

// MessagePermissions returns the permissions of the author of the message
// in the channel in which it was sent.
func (s *State) MessagePermissions(message *Message) (apermissions int64, err error) {
	if s == nil {
		return 0, ErrNilState
	}

	if message.Author == nil || message.Member == nil {
		return 0, ErrMessageIncompletePermissions
	}

	channel, err := s.Channel(message.ChannelID)
	if err != nil {
		return
	}

	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		return
	}

	return memberPermissions(guild, channel, message.Author.ID, message.Member.Roles), nil
}

// UserColor returns the color of a user in a channel.
// While colors are defined at a Guild level, determining for a channel is more useful in message handlers.
// 0 is returned in cases of error, which is the color of @everyone.
// userID    : The ID of the user to calculate the color for.
// channelID   : The ID of the channel to calculate the color for.
func (s *State) UserColor(userID, channelID string) int {
	if s == nil {
		return 0
	}

	channel, err := s.Channel(channelID)
	if err != nil {
		return 0
	}

	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		return 0
	}

	member, err := s.Member(guild.ID, userID)
	if err != nil {
		return 0
	}

	return firstRoleColorColor(guild, member.Roles)
}

// MessageColor returns the color of the author's name as displayed
// in the client associated with this message.
func (s *State) MessageColor(message *Message) int {
	if s == nil {
		return 0
	}

	if message.Member == nil || message.Member.Roles == nil {
		return 0
	}

	channel, err := s.Channel(message.ChannelID)
	if err != nil {
		return 0
	}

	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		return 0
	}

	return firstRoleColorColor(guild, message.Member.Roles)
}

func firstRoleColorColor(guild *Guild, memberRoles []string) int {
	roles := Roles(guild.Roles)
	sort.Sort(roles)

	for _, role := range roles {
		for _, roleID := range memberRoles {
			if role.ID == roleID {
				if role.Color != 0 {
					return role.Color
				}
			}
		}
	}

	for _, role := range roles {
		if role.ID == guild.ID {
			return role.Color
		}
	}

	return 0
}
