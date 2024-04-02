package engine

type Listener interface {
	Update()
	Close()
}
