package core

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/plugin"
)

type pluginEventHookRegistration struct {
	condition client.EventCondition
	id        uint64
}

// RegisterPluginEventHooks replaces the hooks owned by a plugin name.
func (c *Console) RegisterPluginEventHooks(plug plugin.Plugin) {
	if c == nil || c.Server == nil || plug == nil {
		return
	}
	c.UnregisterPluginEventHooks(plug)
	if c.pluginEventHooks == nil {
		c.pluginEventHooks = make(map[plugin.Plugin][]pluginEventHookRegistration)
	}
	for condition, hook := range plug.GetEvents() {
		id := c.AddEventHook(condition, hook)
		c.pluginEventHooks[plug] = append(c.pluginEventHooks[plug], pluginEventHookRegistration{
			condition: condition,
			id:        id,
		})
	}
}

// UnregisterPluginEventHooks removes only hooks owned by the given plugin.
func (c *Console) UnregisterPluginEventHooks(plug plugin.Plugin) {
	if c == nil || c.Server == nil || c.pluginEventHooks == nil {
		return
	}
	for _, registration := range c.pluginEventHooks[plug] {
		c.RemoveEventHookByID(registration.condition, registration.id)
	}
	delete(c.pluginEventHooks, plug)
}
