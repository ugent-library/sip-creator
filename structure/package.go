package structure

type Package struct {
	Root                    *Entity
	PremisFile              *File
	MetsFile                *File
	DescriptiveFiles        []*File
	RepresentationMetsFiles []*File
}

func (p *Package) AddRootEntity(e *Entity) {
	p.Root = e
}

func (p *Package) AddPremisFile(f *File) {
	p.PremisFile = f
}

func (p *Package) AddMetsFile(f *File) {
	p.MetsFile = f
}

func (p *Package) AddDescriptiveFile(f *File) {
	p.DescriptiveFiles = append(p.DescriptiveFiles, f)
}

func NewPackage() *Package {
	return &Package{}
}
