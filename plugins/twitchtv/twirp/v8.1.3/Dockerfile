# syntax=docker/dockerfile:1.4
FROM golang:1.19.2-bullseye AS build
WORKDIR /tmp
RUN git clone --depth 1 https://github.com/twitchtv/twirp.git
WORKDIR /tmp/twirp
COPY --link . /tmp/twirp

RUN git apply <<'EOF'
diff --git a/protoc-gen-twirp/command_line.go b/protoc-gen-twirp/command_line.go
index 8024a1a..9e1abb2 100644
--- a/protoc-gen-twirp/command_line.go
+++ b/protoc-gen-twirp/command_line.go
@@ -23,6 +23,8 @@ type commandLineParams struct {
 	paths        string            // paths flag, used to control file output directory.
 	module       string            // module flag, Go import path prefix that is removed from the output filename.
 	importPrefix string            // prefix added to imported package file names.
+
+	separatePackage bool // write generated files to a separate twirp package
 }
 
 // parseCommandLineParams breaks the comma-separated list of key=value pairs
@@ -72,6 +74,8 @@ func parseCommandLineParams(parameter string) (*commandLineParams, error) {
 		case k == "module":
 			clp.module = v
 
+		case k == "separate_package":
+			clp.separatePackage = (strings.ToLower(v) == "true")
 		// Deprecated, but may still be useful when working with old versions of protoc-gen-go
 		case k == "import_prefix":
 			clp.importPrefix = v
diff --git a/protoc-gen-twirp/generator.go b/protoc-gen-twirp/generator.go
index 601f80f..29122ac 100644
--- a/protoc-gen-twirp/generator.go
+++ b/protoc-gen-twirp/generator.go
@@ -51,6 +51,7 @@ type twirp struct {
 	// Package output:
 	sourceRelativePaths bool // instruction on where to write output files
 	modulePrefix        string
+	separatePackage     bool
 
 	// Package naming:
 	genPkgName          string // Name of the package that we're generating
@@ -86,6 +87,7 @@ func (t *twirp) Generate(in *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorR
 	t.importMap = params.importMap
 	t.sourceRelativePaths = params.paths == "source_relative"
 	t.modulePrefix = params.module
+	t.separatePackage = params.separatePackage
 
 	t.genFiles = gen.FilesToGenerate(in)
 
@@ -116,6 +118,9 @@ func (t *twirp) Generate(in *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorR
 		gen.Fail(err.Error())
 	}
 	t.genPkgName = genPkgName
+	if t.separatePackage {
+		t.genPkgName += "twirp"
+	}
 
 	// We also need to figure out the fully import path of the package we're
 	// generating. It's possible to import proto definitions from different .proto
@@ -127,7 +132,7 @@ func (t *twirp) Generate(in *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorR
 	// Next, we need to pick names for all the files that are dependencies.
 	for _, f := range in.ProtoFile {
 		// Is this is a file we are generating? If yes, it gets the shared package name.
-		if fileDescSliceContains(t.genFiles, f) {
+		if fileDescSliceContains(t.genFiles, f) && !t.separatePackage {
 			t.fileToGoPackageName[f] = t.genPkgName
 			continue
 		}
@@ -137,7 +142,7 @@ func (t *twirp) Generate(in *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorR
 		// shared package name too.
 		if genPkgImportPath != "" {
 			importPath, _, _ := goPackageOption(f)
-			if importPath == genPkgImportPath {
+			if importPath == genPkgImportPath && !t.separatePackage {
 				t.fileToGoPackageName[f] = t.genPkgName
 				continue
 			}
@@ -241,7 +246,15 @@ func (t *twirp) generate(file *descriptor.FileDescriptorProto) *plugin.CodeGener
 
 	t.generateFileDescriptor(file)
 
-	resp.Name = proto.String(t.goFileName(file))
+	fileName := t.goFileName(file)
+	if t.separatePackage {
+		fileName = path.Join(
+			path.Dir(fileName),
+			t.genPkgName,
+			path.Base(fileName),
+		)
+	}
+	resp.Name = proto.String(fileName)
 	resp.Content = proto.String(t.formattedOutput())
 	t.output.Reset()
 
@@ -324,7 +337,14 @@ func (t *twirp) generateImports(file *descriptor.FileDescriptorProto) {
 				importPath = t.importPrefix + importPath
 
 				pkg := t.goPackageName(def.File)
-				if pkg != t.genPkgName {
+				if !t.separatePackage {
+					if pkg != t.genPkgName {
+						deps[pkg] = strconv.Quote(importPath)
+					}
+				} else {
+					if pkg == t.genPkgName {
+						pkg = strings.TrimSuffix(pkg, "twirp")
+					}
 					deps[pkg] = strconv.Quote(importPath)
 				}
 			}
EOF

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod tidy && \
    CGO_ENABLED=0 go build -mod=mod -ldflags "-s -w" -o protoc-gen-twirp-go ./protoc-gen-twirp


FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /tmp/twirp/protoc-gen-twirp-go .
USER nobody
ENTRYPOINT [ "/protoc-gen-twirp-go" ]
