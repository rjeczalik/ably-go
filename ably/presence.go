package ably

type Presence struct {
	client  *RestClient
	channel *RestChannel
}

// Get gives the channel's presence messages according to the given parameters.
// The returned resource can be inspected for the presence messages via
// the PresenceMessages() method.
func (p *Presence) Get(params *PaginateParams) (*PaginatedResource, error) {
	path := "/channels/" + p.channel.Name + "/presence"
	return NewPaginatedResource(presMsgType, path, params, query(p.client.Get))
}

// History gives the channel's presence messages history according to the given
// parameters. The returned resource can be inspected for the presence messages
// via the PresenceMessages() method.
func (p *Presence) History(params *PaginateParams) (*PaginatedResource, error) {
	path := "/channels/" + p.channel.Name + "/presence/history"
	return NewPaginatedResource(presMsgType, path, params, query(p.client.Get))
}
