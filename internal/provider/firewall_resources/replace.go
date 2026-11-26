package firewall_resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func withRequiresReplace(attribute schema.Attribute) schema.Attribute {
	switch value := attribute.(type) {
	case schema.StringAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, stringplanmodifier.RequiresReplace())
		}
		return value
	case schema.BoolAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, boolplanmodifier.RequiresReplace())
		}
		return value
	case schema.Int64Attribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, int64planmodifier.RequiresReplace())
		}
		return value
	case schema.Float64Attribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, float64planmodifier.RequiresReplace())
		}
		return value
	case schema.NumberAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, numberplanmodifier.RequiresReplace())
		}
		return value
	default:
		return collectionRequiresReplace(attribute)
	}
}

func collectionRequiresReplace(attribute schema.Attribute) schema.Attribute {
	switch value := attribute.(type) {
	case schema.ListAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, listplanmodifier.RequiresReplace())
		}
		return value
	case schema.ListNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, listplanmodifier.RequiresReplace())
		}
		return value
	case schema.SetAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, setplanmodifier.RequiresReplace())
		}
		return value
	case schema.SetNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, setplanmodifier.RequiresReplace())
		}
		return value
	case schema.MapAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, mapplanmodifier.RequiresReplace())
		}
		return value
	case schema.MapNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, mapplanmodifier.RequiresReplace())
		}
		return value
	case schema.SingleNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, objectplanmodifier.RequiresReplace())
		}
		return value
	}
	return attribute
}
