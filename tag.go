package tag

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

const (
	DEFAULT_TAG = "valid:\"-\""
)

func init() {
	generator.RegisterPlugin(new(tag))
}

type tag struct {
	gen *generator.Generator // the original generater

	file     *generator.FileDescriptor // The file we are compiling now.
	comments map[string]map[string]string
}

// Name returns the name of this plugin, "tag".
func (t *tag) Name() string {
	return "tag"
}

// Init initializes the plugin.
func (t *tag) Init(g *generator.Generator) {
	t.gen = g
	t.comments = map[string]map[string]string{}
}

func (t *tag) Generate(file *generator.FileDescriptor) {
	t.file = t.gen.FileOf(file.FileDescriptorProto)

	// get each comment and store them into t.comments
	for i, message := range t.file.GetMessageType() {
		path := []int32{4, int32(i)}

		for j, nestedMessage := range message.GetNestedType() {
			nestedPath := append(path, 3, int32(j))

			t.addFieldComment(nestedPath, nestedMessage)
		}

		t.addFieldComment(path, message)
	}

	// use t.comments to rewrite the generated go stub files
	currentStub := bytes.NewBuffer(t.gen.Buffer.Bytes())
	newStub := bytes.NewBuffer([]byte{})
	reader := bufio.NewReader(currentStub)
	var comment bool
	var structName string
	for {
		line, _, err := reader.ReadLine()

		// the for loop exists here
		if err != nil {
			newStub.WriteString("\n")
			break
		}

		lineContent := string(line)

		// handle the comments in stub file
		if strings.HasPrefix(strings.TrimSpace(lineContent), "/*") {
			comment = true
		}

		// if the comment block ends, then just write it and
		// handle another line
		if comment && strings.Contains(lineContent, "*/") {
			comment = false
			newStub.Write(line)
			newStub.WriteString("\n")
			continue
		}

		// if still in comment, just write anything and
		// handle another line
		if comment {
			newStub.Write(line)
			newStub.WriteString("\n")
			continue
		}

		if structName == "" {
			structName = getStructName(lineContent)

			if _, ok := t.comments[structName]; !ok {
				structName = ""
			}

			newStub.Write(line)
			newStub.WriteString("\n")
			continue
		}

		// if code goes into here, means the structName must be
		// in t.comments, we can just replace the tag for each field

		// the struct ends
		if strings.HasPrefix(strings.TrimSpace(lineContent), "}") {
			structName = ""
			newStub.Write(line)
			newStub.WriteString("\n")
			continue
		}

		// begin to replace tag
		fieldName := getFieldName(lineContent)
		tagComment := t.comments[structName][toLowerFirst(fieldName)]
		newLine := insertTag(lineContent, tagComment)
		newStub.WriteString(newLine + "\n")
	}

	t.gen.Reset()
	t.gen.Write(newStub.Bytes())
}

func (t *tag) GenerateImports(file *generator.FileDescriptor) {}

func (t *tag) addFieldComment(path []int32, message *descriptor.DescriptorProto) {
	messageName := message.GetName()
	sourceInfo := t.file.GetSourceCodeInfo()
	for i, field := range message.GetField() {
		fieldPath := append(path, 2, int32(i))
		comment := getTrailingComment(fieldPath, sourceInfo)

		if _, ok := t.comments[messageName]; !ok {
			t.comments[messageName] = map[string]string{}
		}
		t.comments[messageName][field.GetName()] = comment
	}
}

// getTrailingComment will iterate through sourceInfo and get
// the trailing comment via path
func getTrailingComment(path []int32, sourceInfo *descriptor.SourceCodeInfo) string {
	for _, loc := range sourceInfo.GetLocation() {
		if isSamePath(loc.GetPath(), path) {
			comment := strings.TrimSpace(loc.GetTrailingComments())
			if comment != "" {
				return comment
			}
		}
	}

	return DEFAULT_TAG
}

// isSamePath actually is just a slice equality check function
func isSamePath(a, b []int32) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func getStructName(line string) string {
	re := regexp.MustCompile("type (.+) struct {")
	match := re.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func getFieldName(line string) string {
	re := regexp.MustCompile("\t(.+)\t.+\t.+")
	match := re.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1]
	}

	return ""
}

// CammelCase to Lower Camel Case
func toLowerFirst(str string) string {
	if str == "" {
		return str
	}

	if len(str) == 1 {
		return strings.ToLower(str[:1])
	}

	return strings.ToLower(str[:1]) + str[1:]
}

// insert a tag into field line
func insertTag(line, tag string) string {
	re := regexp.MustCompile("(`)$")
	newLine := re.ReplaceAllString(line, " "+tag+"$1")

	return newLine
}
