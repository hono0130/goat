package protobuf

type protobufGenerator struct {
	analyzer *typeAnalyzer
	writer   *fileWriter
	opts     GenerateOptions
}

func newProtobufGenerator(opts GenerateOptions) *protobufGenerator {
	return &protobufGenerator{
		analyzer: newTypeAnalyzer(),
		writer:   newFileWriter(opts.OutputDir, opts.PackageName, opts.GoPackage),
		opts:     opts,
	}
}

func (g *protobufGenerator) generateFromSpecs(specs ...AbstractProtobufServiceSpec) error {
	definitions := g.analyzer.analyzeSpecs(specs...)
	return g.writer.writeProtoFile(g.opts.Filename, definitions)
}
