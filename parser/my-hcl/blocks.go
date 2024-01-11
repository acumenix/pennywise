package my_hcl

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"strings"
)

type Block struct {
	Type        string
	Labels      []string
	Body        hcl.Body
	ChildBlocks []Block
	Attributes  []Attribute
}

var (
	terraformSchema = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type: "terraform",
			},
			{
				Type:       "provider",
				LabelNames: []string{"name"},
			},
			{
				Type:       "variable",
				LabelNames: []string{"name"},
			},
			{
				Type: "locals",
			},
			{
				Type:       "output",
				LabelNames: []string{"name"},
			},
			{
				Type:       "module",
				LabelNames: []string{"name"},
			},
			{
				Type:       "resource",
				LabelNames: []string{"type", "name"},
			},
			{
				Type:       "data",
				LabelNames: []string{"type", "name"},
			},
		},
	}
)
var (
	missingAttributeDiagnostic        = "Unsupported attribute"
	valueDoesNotHaveAnyIndices        = "Invalid index"
	valueIsNonIterableDiagnostic      = "Iteration over non-iterable value"
	invalidFunctionArgumentDiagnostic = "Invalid function argument"
)

func getFileBlocks(file *hcl.File) ([]Block, error) {
	contents, _, diags := file.Body.PartialContent(terraformSchema)
	if diags.HasErrors() {
		return nil, diags
	}
	myBlocks, err := makeBlocks(&contents.Blocks, nil)
	if err != nil {
		return nil, err
	}
	return myBlocks, nil
}

func makeBlocks(blocks *hcl.Blocks, childBlocks *hclsyntax.Blocks) ([]Block, error) {
	var myBlocks []Block
	if blocks != nil {
		for _, b := range *blocks {
			if body, ok := b.Body.(*hclsyntax.Body); ok {
				childBlocks, err := makeBlocks(nil, &body.Blocks)
				if err != nil {
					return nil, err
				}
				attributes := make(hcl.Attributes)
				for _, a := range body.Attributes {
					attributes[a.Name] = a.AsHCLAttribute()
				}
				myBlocks = append(myBlocks, Block{
					Type:        b.Type,
					Labels:      b.Labels,
					Body:        b.Body,
					ChildBlocks: childBlocks,
					Attributes:  BuildAttributes(attributes),
				})
			}
		}
	} else if childBlocks != nil {
		for _, b := range *childBlocks {
			childBlocks, err := makeBlocks(nil, &b.Body.Blocks)
			if err != nil {
				return nil, err
			}
			attributes, diags := b.Body.JustAttributes()
			if diags.HasErrors() {
				return nil, diags
			}
			myBlocks = append(myBlocks, Block{
				Type:        b.Type,
				Labels:      b.Labels,
				Body:        b.Body,
				ChildBlocks: childBlocks,
				Attributes:  BuildAttributes(attributes),
			})

		}
	}

	return myBlocks, nil
}

func (b Block) MakeMapStructure(mappedBlocks map[string]interface{}) (map[string]interface{}, error) {
	mapStructure := make(map[string]interface{})
	for _, attr := range b.Attributes {
		val, err := attr.Value(mappedBlocks)
		if err != nil {
			return nil, err
		}
		switch val.(type) {
		case int64, int32, int:
			mapStructure[attr.Name] = val.(int64)
		case string:
			mapStructure[attr.Name] = val.(string)
		case bool:
			mapStructure[attr.Name] = val.(bool)
		case Block:
			blockValues, err := val.(Block).MakeMapStructure(mappedBlocks)
			if err != nil {
				return nil, err
			}
			mapStructure[attr.Name] = blockValues
		default:
			return nil, fmt.Errorf("value type not implemented")
		}
	}
	for _, childBlock := range b.ChildBlocks {
		var blockName string
		if len(childBlock.Labels) > 0 {
			blockName = fmt.Sprintf("%s.%s", childBlock.Type, strings.Join(childBlock.Labels, "."))
		} else {
			blockName = childBlock.Type
		}
		mappedChildBlock, err := childBlock.MakeMapStructure(mappedBlocks)
		if err != nil {
			return nil, err
		}
		mapStructure[blockName] = mappedChildBlock
	}
	return mapStructure, nil
}
