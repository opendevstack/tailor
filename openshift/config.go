package openshift

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/cli"
	"github.com/xeipuuv/gojsonpointer"
	"regexp"
	"strconv"
	"strings"
)

var (
	blacklistedKeys = []string{
		"metadata/annotations/(kubectl.kubernetes.io|pv.kubernetes.io|openshift.io~1host.generated|openshift.io~1generated-by|openshift.io~1image.dockerRepositoryCheck)",
		"metadata/generation$",
		"spec/tags$",
		"status$",
		"spec/volumeName$",
	}
	emptyMapKeys = map[string]string{
		"metadata$": "annotations",
	}
	modifiedKeys = []string{
		"spec/template/spec/containers/[0-9]+/image$",
	}
)

type Config struct {
	Raw              []byte
	Processed        map[string]interface{}
	NameRegex        string
	PointersToInit   []string
	PointersToDelete []string
	PointersToReset  map[string]string
	ItemPointers     []string
	Items            []*ResourceItem
}

func NewConfigFromTemplate(input []byte) *Config {
	c := &Config{
		Raw:             input,
		NameRegex:       "/objects/[0-9]+/metadata/name",
		PointersToReset: make(map[string]string),
	}
	c.Process()
	return c
}

func NewConfigFromList(input []byte) *Config {
	c := &Config{
		Raw:             input,
		NameRegex:       "/items/[0-9]+/metadata/name",
		PointersToReset: make(map[string]string),
	}
	c.Process()
	return c
}

func (c *Config) Process() {
	if len(c.Raw) == 0 {
		return
	}

	var f interface{}
	yaml.Unmarshal(c.Raw, &f)

	m := f.(map[string]interface{})

	pointer := ""

	c.walkMap(m, pointer)

	for _, p := range c.PointersToDelete {
		deletePointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _ = deletePointer.Delete(m)
	}

	for _, p := range c.PointersToInit {
		initPointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _, err := initPointer.Get(m)
		if err != nil {
			initPointer.Set(m, make(map[string]interface{}))
		}
	}

	// If there is an annotation, copy its value into the config, otherwise
	// copy the config value into the annotation.
	for configPath, annotationPath := range c.PointersToReset {
		annotationPointer, _ := gojsonpointer.NewJsonPointer(annotationPath)
		configPointer, _ := gojsonpointer.NewJsonPointer(configPath)
		annotationValue, _, err := annotationPointer.Get(m)
		if err == nil {
			configPointer.Set(m, annotationValue)
		} else {
			configValue, _, _ := configPointer.Get(m)
			annotationPointer.Set(m, configValue)
		}
	}

	c.Processed = m

	c.Items = c.collectItems()
}

func (c *Config) collectItems() []*ResourceItem {
	items := []*ResourceItem{}
	for _, itemPointer := range c.ItemPointers {
		kindPointer := itemPointer + "/kind"
		namePointer := itemPointer + "/metadata/name"
		labelsPointer := itemPointer + "/metadata/labels"
		kp, _ := gojsonpointer.NewJsonPointer(kindPointer)
		kind, _, _ := kp.Get(c.Processed)
		cp, _ := gojsonpointer.NewJsonPointer(itemPointer)
		config, _, _ := cp.Get(c.Processed)
		np, _ := gojsonpointer.NewJsonPointer(namePointer)
		name, _, _ := np.Get(c.Processed)
		lp, _ := gojsonpointer.NewJsonPointer(labelsPointer)
		labels, _, err := lp.Get(c.Processed)
		if err != nil {
			labels = make(map[string]interface{})
		}
		item := &ResourceItem{
			Name:    name.(string),
			Kind:    kind.(string),
			Labels:  labels.(map[string]interface{}),
			Pointer: itemPointer,
			Config:  config,
		}
		items = append(items, item)
	}

	return items
}

func (c *Config) ExtractResources(filter *ResourceFilter) []*ResourceItem {
	items := []*ResourceItem{}
	for _, item := range c.Items {
		if filter.SatisfiedBy(item) {
			items = append(items, item)
		}
	}

	return items
}

func (c *Config) walkMap(m map[string]interface{}, pointer string) {
	for k, v := range m {
		c.handleKeyValue(k, v, pointer)
	}
}

func (c *Config) walkArray(a []interface{}, pointer string) {
	for k, v := range a {
		c.handleKeyValue(k, v, pointer)
	}
}

func (c *Config) handleKeyValue(k interface{}, v interface{}, pointer string) {

	strK := ""
	switch kv := k.(type) {
	case string:
		strK = kv
	case int:
		strK = strconv.Itoa(kv)
	}

	// See https://tools.ietf.org/html/draft-ietf-appsawg-json-pointer-07#section-3.
	relativePointer := strings.Replace(strK, "~", "~0", -1)
	relativePointer = strings.Replace(relativePointer, "/", "~1", -1)
	absolutePointer := pointer + "/" + relativePointer

	for emptyMapKeyPath, keyToInit := range emptyMapKeys {
		matched, _ := regexp.MatchString(emptyMapKeyPath, absolutePointer)
		if matched {
			c.PointersToInit = append(c.PointersToInit, absolutePointer+"/"+keyToInit)
			break
		}
	}

	deletePointer := false
	for _, blacklistedKey := range blacklistedKeys {
		matched, _ := regexp.MatchString(blacklistedKey, absolutePointer)
		if matched {
			alreadyRegisteredForDeletion := false
			for _, p := range c.PointersToDelete {
				if strings.HasPrefix(absolutePointer, p) {
					alreadyRegisteredForDeletion = true
					continue
				}
			}
			if !alreadyRegisteredForDeletion {
				c.PointersToDelete = append(c.PointersToDelete, absolutePointer)
				deletePointer = true
			}
			break
		}
	}

	if !deletePointer {
		nameRegexMatched, _ := regexp.MatchString(c.NameRegex, absolutePointer)
		if nameRegexMatched {
			parts := strings.Split(absolutePointer, "/")
			itemPointer := strings.Join(parts[0:3], "/")
			cli.VerboseMsg(fmt.Sprintf("Detected item %s:%s", absolutePointer, v.(string)))
			c.ItemPointers = append(c.ItemPointers, itemPointer)
		}

		for _, modifiedKey := range modifiedKeys {
			matched, _ := regexp.MatchString(modifiedKey, absolutePointer)
			if matched {
				parts := strings.Split(absolutePointer, "/")
				itemPointer := strings.Join(parts[0:3], "/")
				annotationKey := strings.Replace(absolutePointer, itemPointer+"/", "", -1)
				annotationKey = strings.Replace(annotationKey, "/", ".", -1)
				c.PointersToReset[absolutePointer] = itemPointer + "/metadata/annotations/original-values.tailor.io~1" + annotationKey
				break
			}
		}
	}

	switch vv := v.(type) {
	case []interface{}:
		c.walkArray(vv, absolutePointer)
	case map[string]interface{}:
		c.walkMap(vv, absolutePointer)
	}
}
