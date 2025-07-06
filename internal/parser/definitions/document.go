package definitions

type Document struct {
	Source       *DocumentDef
	Meta         *MetaBlock
	Vars         *Vars
	RequiredVars []string

	DataBlocks        []*DataBlock
	ContentTreeBlocks []ContentTreeBlock
	FormatBlocks      []*FormatBlock
	PublishBlocks     []*PublishBlock
}

func (b *Document) isRefTargetBlock()   {}
func (b *Document) GetSourceKind() string {
	return b.Source.Kind()
}
var _ RefTargetBlock = (*Document)(nil)
