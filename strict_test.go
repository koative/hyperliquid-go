package hyperliquid

// Every test in this package decodes strictly: a response field that the
// types do not model fails the test instead of being dropped silently.
func init() { strictDecoding = true }
