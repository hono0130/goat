package openapi

type generator struct {
	analyzer *schemaAnalyzer
	writer   *specWriter
	opts     GenerateOptions
}

func newGenerator(opts GenerateOptions) *generator {
	return &generator{
		analyzer: newSchemaAnalyzer(),
		writer:   newSpecWriter(opts.OutputDir, opts.Title, opts.Version),
		opts:     opts,
	}
}

func (g *generator) generateFromSpecs(specs ...AbstractServiceSpec) error {
	definitions := g.analyzer.analyzeSpecs(specs...)
	return g.writer.writeOpenAPIFile(g.opts.Filename, definitions)
}
