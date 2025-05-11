package vpn

type Device struct {
	Addresses                 []string `json:"addresses"`
	Id                        string   `json:"id"`
	NodeId                    string   `json:"nodeId"`
	User                      string   `json:"user"`
	Name                      string   `json:"name"`
	Hostname                  string   `json:"hostname"`
	ClientVersion             string   `json:"clientVersion"`
	IsUpdateAvailable         bool     `json:"updateAvailable"`
	OS                        string   `json:"os"`
	Created                   string   `json:"created"`
	LastSeen                  string   `json:"lastSeen"`
	IsKeyExpiryDisabled       bool     `json:"keyExpiryDisable"`
	ExpiresAt                 string   `json:"expires"`
	IsAuthorized              bool     `json:"authorized"`
	IsExternal                bool     `json:"isExternal"`
	MachineKey                string   `json:"machineKey"`
	NodeKey                   string   `json:"nodeKey"`
	BlocksIncomingConnections bool     `json:"blocksIncomingConnections"`
	TailnetLockKey            string   `json:"tailnetLockKey"`
	TailnetLockError          string   `json:"tailnetLockError"`
	Tags                      []string `json:"tags"`
}

type NetPolicy struct {
	Groups    map[string][]string `json:"groups"`
	TagOwners map[string][]string `json:"tagOwners"`
	Hosts     map[string]string   `json:"hosts"`
	ACLs      []ACL               `json:"acls"`
	SshACLs   []ACL               `json:"ssh"`
}

type ACL struct {
	Action      string   `json:"action"`
	Source      []string `json:"src"`
	Destination []string `json:"dst"`
	Users       []string `json:"users"`
}
