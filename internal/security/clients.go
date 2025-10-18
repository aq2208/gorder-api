package security

// In-memory client registry (replace with DB/config later)
type Client struct {
	ID      string
	Secret  string
	Perms   []string // e.g. {"orders.read","orders.write"}
	Enabled bool
}

var Clients = map[string]Client{
	"simulated-client": {ID: "simulated-client", Secret: "simulated-client-secret", Perms: []string{"orders.read", "orders.write"}, Enabled: true},
	"svc-order-gw":     {ID: "svc-order-gw", Secret: "gw-secret", Perms: []string{"orders.read", "orders.write"}, Enabled: true},
	"svc-analytics":    {ID: "svc-analytics", Secret: "ana-secret", Perms: []string{"orders.read"}, Enabled: true},
}
