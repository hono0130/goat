package openapi

type openAPIGenerator struct {
	analyzer *schemaAnalyzer
	writer   *specWriter
	opts     GenerateOptions
}

func newOpenAPIGenerator(opts GenerateOptions) *openAPIGenerator {
	return &openAPIGenerator{
		analyzer: newSchemaAnalyzer(),
		writer:   newSpecWriter(opts.OutputDir, opts.Title, opts.Version),
		opts:     opts,
	}
}

func (g *openAPIGenerator) generateFromSpecs(specs ...AbstractOpenAPIServiceSpec) error {
	definitions := g.analyzer.analyzeSpecs(specs...)
	return g.writer.writeOpenAPIFile(g.opts.Filename, definitions)
}
