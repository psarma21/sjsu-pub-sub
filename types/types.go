package types

type User struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

type Group struct {
	GroupName  string   `bson:"groupname"`
	Creator    string   `bson:"creator"`
	GroupMates []string `bson:"groupmates"`
	Posts      []Post   `bson:"posts"`
}

type Post struct {
	Author string `bson:"author"`
	Group  string `bson:"group"`
	Body   string `bson:"body"`
}

type AuthMessage struct {
	Username string `json:"username"`
	Port     string `json:"port"`
}

// message sent via TCP from server to client or client to client
type GossipMessage struct {
	Id           int      `json:"id"`
	Body         string   `json:"body"`
	ConnsToWrite []string `json:"connsToWrite"`
}
