package protobuf

type generator struct {
	analyzer *typeAnalyzer
	writer   *fileWriter
	opts     GenerateOptions
}

func newGenerator(opts GenerateOptions) *generator {
	return &generator{
		analyzer: newTypeAnalyzer(),
		writer:   newFileWriter(opts.OutputDir, opts.PackageName, opts.GoPackage),
		opts:     opts,
	}
}

func (g *generator) generateFromSpecs(specs ...AbstractServiceSpec) error {
	definitions := g.analyzer.analyzeSpecs(specs...)
	return g.writer.writeProtoFile(g.opts.Filename, definitions)
}
