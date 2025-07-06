package definitions

type RefTargetBlock interface {
	isRefTargetBlock()
	GetSourceKind() string
}
