package models

import "time"

type Info struct {
	Version string `json:"version"`

	Build struct {
		NodeVersion string `json:"nodeVersion"`
		Arch        string `json:"arch"`
		Platform    string `json:"platform"`
		Cpus        int    `json:"cpus"`
	} `json:"build"`

	Commit struct {
		Hash    string `json:"hash"`
		Date    string `json:"date"`
		Author  string `json:"author"`
		Subject string `json:"subject"`
		Tag     string `json:"tag"`
		Branch  string `json:"branch"`
	} `json:"commit"`
}

type Pagination struct {
	Count  int `json:"count"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type Directory struct {
	Result []struct {
		ID        string    `json:"_id"`
		CreatedAt time.Time `json:"createdAt"`
		Emails    []struct {
			Address  string `json:"address"`
			Verified bool   `json:"verified"`
		} `json:"emails"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"result"`

	Pagination
}

type Spotlight struct {
	Users []User    `json:"users"`
	Rooms []Channel `json:"rooms"`
}

type Statistics struct {
	ID       string `json:"_id"`
	UniqueID string `json:"uniqueId"`
	Version  string `json:"version"`

	ActiveUsers    int `json:"activeUsers"`
	NonActiveUsers int `json:"nonActiveUsers"`
	OnlineUsers    int `json:"onlineUsers"`
	AwayUsers      int `json:"awayUsers"`
	OfflineUsers   int `json:"offlineUsers"`
	TotalUsers     int `json:"totalUsers"`

	TotalRooms                int `json:"totalRooms"`
	TotalChannels             int `json:"totalChannels"`
	TotalPrivateGroups        int `json:"totalPrivateGroups"`
	TotalDirect               int `json:"totalDirect"`
	TotlalLivechat            int `json:"totlalLivechat"`
	TotalMessages             int `json:"totalMessages"`
	TotalChannelMessages      int `json:"totalChannelMessages"`
	TotalPrivateGroupMessages int `json:"totalPrivateGroupMessages"`
	TotalDirectMessages       int `json:"totalDirectMessages"`
	TotalLivechatMessages     int `json:"totalLivechatMessages"`

	InstalledAt          time.Time `json:"installedAt"`
	LastLogin            time.Time `json:"lastLogin"`
	LastMessageSentAt    time.Time `json:"lastMessageSentAt"`
	LastSeenSubscription time.Time `json:"lastSeenSubscription"`

	Os struct {
		Type     string    `json:"type"`
		Platform string    `json:"platform"`
		Arch     string    `json:"arch"`
		Release  string    `json:"release"`
		Uptime   int       `json:"uptime"`
		Loadavg  []float64 `json:"loadavg"`
		Totalmem int64     `json:"totalmem"`
		Freemem  int       `json:"freemem"`
		Cpus     []struct {
			Model string `json:"model"`
			Speed int    `json:"speed"`
			Times struct {
				User int `json:"user"`
				Nice int `json:"nice"`
				Sys  int `json:"sys"`
				Idle int `json:"idle"`
				Irq  int `json:"irq"`
			} `json:"times"`
		} `json:"cpus"`
	} `json:"os"`

	Process struct {
		NodeVersion string  `json:"nodeVersion"`
		Pid         int     `json:"pid"`
		Uptime      float64 `json:"uptime"`
	} `json:"process"`

	Deploy struct {
		Method   string `json:"method"`
		Platform string `json:"platform"`
	} `json:"deploy"`

	Migration struct {
		ID       string    `json:"_id"`
		Version  int       `json:"version"`
		Locked   bool      `json:"locked"`
		LockedAt time.Time `json:"lockedAt"`
		BuildAt  time.Time `json:"buildAt"`
	} `json:"migration"`

	InstanceCount int       `json:"instanceCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"_updatedAt"`
}

type StatisticsInfo struct {
	Statistics Statistics `json:"statistics"`
}

type StatisticsList struct {
	Statistics []Statistics `json:"statistics"`

	Pagination
}
