package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

// Ensure the implementations satisfy the expected interfaces.
var (
	_ resource.Resource = &resourceNEAP{}
	_ tfmodel           = &neapModel{}
)

// =================== [ TF Models ] ===================

// =================== [ TF Models / NEAP ] ===================

type neapModel struct {
	ID                      types.String `tfsdk:"id"`
	Title                   types.String `tfsdk:"title"`
	Description             types.String `tfsdk:"description"`
	Disabled                types.Bool   `tfsdk:"disabled"`
	Priority                types.Int64  `tfsdk:"priority"`
	SplitByField            types.Set    `tfsdk:"split_by_field"`
	EntityFactorEnabled     types.Bool   `tfsdk:"entity_factor_enabled"` //TODO: remove ???
	RunTimeBasedActionsOnce types.Bool   `tfsdk:"run_time_based_actions_once"`
	ServiceTopologyEnabled  types.Bool   `tfsdk:"service_topology_enabled"` //TODO: remove ???
	GroupTitle              types.String `tfsdk:"group_title"`
	GroupDescription        types.String `tfsdk:"group_description"`
	GroupSeverity           types.String `tfsdk:"group_severity"`
	GroupAssignee           types.String `tfsdk:"group_assignee"`
	GroupStatus             types.String `tfsdk:"group_status"`
	GroupInstruction        types.String `tfsdk:"group_instruction"`
	GroupCustomInstruction  types.String `tfsdk:"group_custom_instruction"`
	GroupDashboard          types.String `tfsdk:"group_dashboard"`
	GroupDashboardContext   types.String `tfsdk:"group_dashboard_context"`

	BreakingCriteria *neapCriteriaModel `tfsdk:"breaking_criteria"`
	FilterCriteria   *neapCriteriaModel `tfsdk:"filter_criteria"`

	Rules []neapRuleModel `tfsdk:"rule"`
}

func (n neapModel) objectype() string {
	return "notable_event_aggregation_policy"
}

func (n neapModel) title() string {
	return n.Title.String()
}

// func (n *neapModel) normalize() {
// 	if n.BreakingCriteria != nil {
// 		n.BreakingCriteria.normalize(neapCriteriaTypeBreaking)
// 	}
// 	if n.FilterCriteria != nil {
// 		n.FilterCriteria.normalize(neapCriteriaTypeFilter)
// 	}
// }

// =================== [ TF Models / NEAP / Criteria] ===================

type neapCriteriaModel struct {
	Condition         types.String                               `tfsdk:"condition"`
	Clause            []neapCriteriaClauseModel                  `tfsdk:"clause"`
	Pause             []neapCriteriaClausePauseModel             `tfsdk:"pause"`
	Duration          []neapCriteriaClauseDurationModel          `tfsdk:"duration"`
	NotableEventCount []neapCriteriaClauseNotableEventCountModel `tfsdk:"notable_event_count"`
	BreakingCriteria  []neapCriteriaClauseBreakingCriteriaModel  `tfsdk:"breaking_criteria"`
}

type neapCriteriaType int

const (
	neapCriteriaTypeBreaking = iota
	neapCriteriaTypeFilter
	neapCriteriaTypeActivation
)

// func (c *neapCriteriaModel) normalize(criteriaType neapCriteriaType) {
// 	if criteriaType == neapCriteriaTypeActivation {
// 		c.Condition = types.StringValue("AND")
// 	} else {
// 		c.Condition = types.StringValue("OR")

// 	}
// }

func (c *neapCriteriaModel) validate(criteriaType neapCriteriaType) (diags diag.Diagnostics) {
	// TODO: remove this, as long as we don't need to make custom validations.
	// conflicts are now validated w/ schema validators
	switch criteriaType {
	case neapCriteriaTypeBreaking:
		if len(c.BreakingCriteria) > 0 {
			diags.AddError("NEAP: Unsupported criteria", fmt.Sprintf("unsupported: %s item cannot be child of %s", "breaking_criteria", "breaking_criteria"))
		}
	case neapCriteriaTypeFilter:
		if len(c.Pause) > 0 {
			diags.AddError("NEAP: Unsupported criteria", fmt.Sprintf("unsupported: %s item cannot be child of %s", "pause", "filter_criteria"))
		}
		if len(c.Duration) > 0 {
			diags.AddError("NEAP: Unsupported criteria", fmt.Sprintf("unsupported: %s item cannot be child of %s", "duration", "filter_criteria"))
		}
		if len(c.NotableEventCount) > 0 {
			diags.AddError("NEAP: Unsupported criteria", fmt.Sprintf("unsupported: %s item cannot be child of %s", "notable_event_count", "filter_criteria"))
		}
		if len(c.BreakingCriteria) > 0 {
			diags.AddError("NEAP: Unsupported criteria", fmt.Sprintf("unsupported: %s item cannot be child of %s", "breaking_criteria", "filter_criteria"))
		}
	case neapCriteriaTypeActivation:
		break
	default:
		diags.AddError("NEAP: Unexpected Error", "Unexpected error while preparing NEAP criteria api model: invalid criteria type")
	}
	return
}

func (c *neapCriteriaModel) apiModel(criteriaType neapCriteriaType) (criteria map[string]any, diags diag.Diagnostics) {
	if diags = c.validate(criteriaType); diags.HasError() {
		return
	}
	criteria = map[string]any{"items": []any{}, "condition": "OR"}
	if criteriaType == neapCriteriaTypeActivation {
		criteria["condition"] = "AND"
	}

	criteriaItems := []map[string]any{}

	for _, clause := range c.Clause {
		clauseItems := make([]map[string]any, len(clause.NotableEventField))
		for i, field := range clause.NotableEventField {
			clauseItems[i] = map[string]any{
				"type": "notable_event_field",
				"config": map[string]any{
					"field":    field.Field.ValueString(),
					"operator": field.Operator.ValueString(),
					"value":    field.Value.ValueString(),
				},
			}
		}
		criteriaItems = append(criteriaItems, map[string]any{
			"type": "clause",
			"config": map[string]any{
				"condition": clause.Condition.ValueString(),
				"items":     clauseItems,
			},
		})
	}

	for _, pause := range c.Pause {
		criteriaItems = append(criteriaItems, map[string]any{
			"type":   "pause",
			"config": map[string]any{"limit": pause.Limit.ValueInt64()},
		})
	}

	for _, duration := range c.Duration {
		criteriaItems = append(criteriaItems, map[string]any{
			"type":   "duration",
			"config": map[string]any{"limit": duration.Limit.ValueInt64()},
		})
	}

	for _, notableEventCount := range c.NotableEventCount {
		criteriaItems = append(criteriaItems, map[string]any{
			"type": "notable_event_count",
			"config": map[string]any{
				"operator": notableEventCount.Operator.ValueString(),
				"limit":    notableEventCount.Limit.ValueInt64(),
			},
		})
	}

	criteria["items"] = criteriaItems
	return
}

func newNEAPCriteriaFromAPIModel(c map[string]any) (criteria *neapCriteriaModel, diags diag.Diagnostics) {
	criteria = &neapCriteriaModel{}

	if condition, ok := c["condition"]; ok {
		criteria.Condition = types.StringValue(condition.(string))
	}

	items, err := unpackSlice[map[string]any](c["items"])
	if err != nil {
		diags.AddError("NEAP: Invalid Criteria", fmt.Sprintf("Invalid criteria items: %s", err.Error()))
		return
	}

	for _, item := range items {
		itemType := item["type"].(string)
		switch itemType {
		case "notable_event_count":
			config := item["config"].(map[string]any)
			if criteria.NotableEventCount == nil {
				criteria.NotableEventCount = []neapCriteriaClauseNotableEventCountModel{}
			}
			criteria.NotableEventCount = append(criteria.NotableEventCount, neapCriteriaClauseNotableEventCountModel{
				Operator: types.StringValue(config["operator"].(string)),
				Limit:    types.Int64Value(int64(config["limit"].(float64))),
			})
		case "pause":
			config := item["config"].(map[string]any)
			if criteria.Pause == nil {
				criteria.Pause = []neapCriteriaClausePauseModel{}
			}
			criteria.Pause = append(criteria.Pause, neapCriteriaClausePauseModel{
				Limit: types.Int64Value(int64(config["limit"].(float64))),
			})
		case "duration":
			config := item["config"].(map[string]any)
			if criteria.Duration == nil {
				criteria.Duration = []neapCriteriaClauseDurationModel{}
			}
			criteria.Duration = append(criteria.Duration, neapCriteriaClauseDurationModel{
				Limit: types.Int64Value(int64(config["limit"].(float64))),
			})
		case "clause":
			config := item["config"].(map[string]any)
			if criteria.Clause == nil {
				criteria.Clause = []neapCriteriaClauseModel{}
			}
			clause := neapCriteriaClauseModel{
				Condition:         types.StringValue(config["condition"].(string)),
				NotableEventField: []neapCriteriaClauseNotableEventFieldModel{},
			}

			configItems, err := unpackSlice[map[string]any](config["items"])
			if err != nil {
				diags.AddError("NEAP: Invalid Criteria", fmt.Sprintf("Invalid clause items: %s", err.Error()))
				return
			}

			for _, field := range configItems {
				if itemConfig, ok := field["config"].(map[string]any); ok {
					clause.NotableEventField = append(clause.NotableEventField, neapCriteriaClauseNotableEventFieldModel{
						Field:    types.StringValue(itemConfig["field"].(string)),
						Operator: types.StringValue(itemConfig["operator"].(string)),
						Value:    types.StringValue(itemConfig["value"].(string)),
					})
				} else {
					diags.AddError("NEAP: Invalid Criteria", fmt.Sprintf("unsupported notable_event_field criteria: no config found %#v", config))
					return
				}
			}
			criteria.Clause = append(criteria.Clause, clause)
		default:
			diags.AddError("NEAP: Invalid Criteria", fmt.Sprintf("unsupported criteria type: %s", itemType))
		}
	}

	return
}

// =================== [ TF Models / NEAP / Criteria / Clause] ===================

type neapCriteriaClauseModel struct {
	Condition         types.String                               `tfsdk:"condition"`
	NotableEventField []neapCriteriaClauseNotableEventFieldModel `tfsdk:"notable_event_field"`
}

// =================== [ TF Models / NEAP / Criteria / Clause / Notable Event Field ] ===================

type neapCriteriaClauseNotableEventFieldModel struct {
	Field    types.String `tfsdk:"field"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// =================== [ TF Models / NEAP / Criteria / Clause / Pause ] ===================

type neapCriteriaClausePauseModel struct {
	Limit types.Int64 `tfsdk:"limit"`
}

// =================== [ TF Models / NEAP / Criteria / Clause / Duration ] ===================

type neapCriteriaClauseDurationModel struct {
	Limit types.Int64 `tfsdk:"limit"`
}

// =================== [ TF Models / NEAP / Criteria / Clause / Notable Event Count ] ===================

type neapCriteriaClauseNotableEventCountModel struct {
	Operator types.String `tfsdk:"operator"`
	Limit    types.Int64  `tfsdk:"limit"`
}

// =================== [ TF Models / NEAP / Criteria / Clause / Breaking Criteria ] ===================

type neapCriteriaClauseBreakingCriteriaModel struct {
	Config types.String `tfsdk:"config"`
}

// =================== [ TF Models / NEAP / Rule ] ===================

type neapRuleModel struct {
	ID                 types.String           `tfsdk:"id"`
	Description        types.String           `tfsdk:"description"`
	Title              types.String           `tfsdk:"title"`
	Priority           types.Int64            `tfsdk:"priority"`
	ActivationCriteria *neapCriteriaModel     `tfsdk:"activation_criteria"`
	Actions            []neapRuleActionsModel `tfsdk:"actions"`
}

func (r *neapRuleModel) apiModel() (rule map[string]any, diags diag.Diagnostics) {
	var d diag.Diagnostics
	actions := make([]map[string]any, len(r.Actions))
	for i, a := range r.Actions {
		actions[i], d = a.apiModel()
		if diags = append(diags, d...); diags.HasError() {
			return
		}
	}

	activationCriteria, d := r.ActivationCriteria.apiModel(neapCriteriaTypeActivation)
	if diags = append(diags, d...); diags.HasError() {
		return
	}

	var id string
	if r.ID.IsUnknown() || r.ID.ValueString() == "" {
		id, _ = uuid.GenerateUUID()
	} else {
		id = r.ID.ValueString()
	}

	rule = map[string]any{
		"_key":                id,
		"title":               r.Title.ValueString(),
		"description":         r.Description.ValueString(),
		"priority":            r.Priority.ValueInt64(),
		"activation_criteria": activationCriteria,
		"actions":             actions,
	}

	return
}

func NEAPRuleFromAPIModel(r map[string]any) (rule neapRuleModel, diags diag.Diagnostics) {
	activationCriteria, d := newNEAPCriteriaFromAPIModel(r["activation_criteria"].(map[string]any))
	if diags = append(diags, d...); diags.HasError() {
		return
	}

	itsiActions, err := unpackSlice[map[string]any](r["actions"])
	if err != nil {
		diags.AddError("NEAP: Invalid Rule Actions", fmt.Sprintf("Invalid rule actions: %s", err.Error()))
		return
	}

	actions := make([]neapRuleActionsModel, len(itsiActions))
	for i, a := range itsiActions {
		actions[i], d = NEAPRuleActionsFromAPIModel(a)
		if diags = append(diags, d...); diags.HasError() {
			return
		}
	}

	rule = neapRuleModel{
		ID:                 types.StringValue(r["_key"].(string)),
		Title:              types.StringValue(r["title"].(string)),
		Description:        types.StringValue(r["description"].(string)),
		Priority:           types.Int64Value(int64(r["priority"].(float64))),
		ActivationCriteria: activationCriteria,
		Actions:            actions,
	}

	return
}

type neapRuleActionsModel struct {
	Condition types.String               `tfsdk:"condition"`
	Items     []neapRuleActionsItemModel `tfsdk:"item"`
}

func (a *neapRuleActionsModel) apiModel() (action map[string]any, diags diag.Diagnostics) {
	var d diag.Diagnostics
	items := make([]map[string]any, len(a.Items))
	for i, item := range a.Items {
		items[i], d = item.apiModel()
		if diags = append(diags, d...); diags.HasError() {
			return
		}
	}

	action = map[string]any{
		"condition": a.Condition.ValueString(),
		"items":     items,
	}

	return
}

func NEAPRuleActionsFromAPIModel(a map[string]any) (actions neapRuleActionsModel, diags diag.Diagnostics) {
	items, err := unpackSlice[map[string]any](a["items"])
	if err != nil {
		diags.AddError("NEAP: Invalid Rule Actions", fmt.Sprintf("Invalid rule actions items: %s", err.Error()))
	}

	actions.Condition = types.StringValue(a["condition"].(string))

	actions.Items = make([]neapRuleActionsItemModel, len(items))
	for i, item := range items {
		actions.Items[i], diags = NEAPRuleActionsItemFromAPIModel(item)
		if diags.HasError() {
			return
		}
	}

	return
}

type neapStandardAction = string

const (
	neapActionChangeSeverity neapStandardAction = "change_severity"
	neapActionChangeStatus   neapStandardAction = "change_status"
	neapActionChangeOwner    neapStandardAction = "change_owner"
	neapActionComment        neapStandardAction = "comment"

	itsiNeapActionNotableEventChange        = "notable_event_change"
	itsiNeapActionNotableEventComment       = "notable_event_comment"
	itsiNeapActionNotableEventExecuteAction = "notable_event_execute_action"
)

type neapStandardActionTypeAndField struct {
	itsiActionType string
	field          string
}

func neapActionTfToItsiSeverityTransform() map[string]string {
	numericValueByLabel := make(map[string]string)
	for label, info := range util.SeverityMap {
		numericValueByLabel[label] = strconv.Itoa(info.SeverityValue)
	}
	return numericValueByLabel
}

func neapActionTfToItsiStatusTransform() map[string]string {
	numericValueByLabel := make(map[string]string)
	for label, v := range util.StatusInfoMap {
		numericValueByLabel[label] = strconv.Itoa(v)
	}
	return numericValueByLabel
}

var (
	itsiNeapStandardActionChangeSeverity = neapStandardActionTypeAndField{itsiNeapActionNotableEventChange, "severity"}
	itsiNeapStandardActionChangeStatus   = neapStandardActionTypeAndField{itsiNeapActionNotableEventChange, "status"}
	itsiNeapStandardActionChangeOwner    = neapStandardActionTypeAndField{itsiNeapActionNotableEventChange, "owner"}
	itsiNeapStandardActionComment        = neapStandardActionTypeAndField{itsiNeapActionNotableEventComment, ""}

	neapStandardActionTfToItsiValueMapping = map[neapStandardActionTypeAndField]map[string]string{
		itsiNeapStandardActionChangeSeverity: neapActionTfToItsiSeverityTransform(),
		itsiNeapStandardActionChangeStatus:   neapActionTfToItsiStatusTransform(),
	}

	neapStandardActions = map[neapStandardAction]neapStandardActionTypeAndField{
		neapActionChangeSeverity: itsiNeapStandardActionChangeSeverity,
		neapActionChangeStatus:   itsiNeapStandardActionChangeStatus,
		neapActionChangeOwner:    itsiNeapStandardActionChangeOwner,
		neapActionComment:        itsiNeapStandardActionComment,
	}
)

type neapRuleActionsItemModel struct {
	ExecuteOn      types.String `tfsdk:"execute_on"`
	ChangeSeverity types.String `tfsdk:"change_severity"`
	ChangeStatus   types.String `tfsdk:"change_status"`
	ChangeOwner    types.String `tfsdk:"change_owner"`
	Comment        types.String `tfsdk:"comment"`

	Custom []neapRuleActionsCustomActionModel `tfsdk:"custom"`
}

type neapRuleActionsCustomActionModel struct {
	Type   types.String `tfsdk:"type"`
	Config types.String `tfsdk:"config"`
}

func (a *neapRuleActionsItemModel) field(spec neapStandardActionTypeAndField) *types.String {
	switch spec {
	case itsiNeapStandardActionChangeSeverity:
		return &a.ChangeSeverity
	case itsiNeapStandardActionChangeStatus:
		return &a.ChangeStatus
	case itsiNeapStandardActionChangeOwner:
		return &a.ChangeOwner
	case itsiNeapStandardActionComment:
		return &a.Comment
	default:
		panic("unsupported action type")
	}
}

func (a *neapRuleActionsItemModel) apiModel() (item map[string]any, diags diag.Diagnostics) {
	item = map[string]any{
		"execution_criteria": map[string]string{"execute_on": a.ExecuteOn.ValueString()},
	}

	config := make(map[string]any)

	for k, v := range map[neapStandardAction]string{
		neapActionChangeSeverity: a.ChangeSeverity.ValueString(),
		neapActionChangeStatus:   a.ChangeStatus.ValueString(),
		neapActionChangeOwner:    a.ChangeOwner.ValueString(),
		neapActionComment:        a.Comment.ValueString(),
	} {
		if v == "" {
			continue
		}

		var actionTypeAndField neapStandardActionTypeAndField
		var ok bool
		if actionTypeAndField, ok = neapStandardActions[k]; !ok {
			diags.AddError("NEAP: Invalid Standard Action", fmt.Sprintf("unsupported action type: %s", k))
			return
		}

		item["type"] = actionTypeAndField.itsiActionType
		if actionTypeAndField.field != "" {
			config["field"] = actionTypeAndField.field
		}

		transform := neapStandardActionTfToItsiValueMapping[actionTypeAndField]
		if transform == nil {
			config["value"] = v
		} else {
			if value, ok := transform[v]; ok {
				config["value"] = value
			} else {
				diags.AddError("NEAP: Invalid Standard Action", fmt.Sprintf("unsupported value for action type: %s", v))
			}
		}

		break
	}

	for _, custom := range a.Custom {
		action := custom.Type.ValueString()
		item["type"] = itsiNeapActionNotableEventExecuteAction
		config["name"] = action

		var tfActionConfig map[string]any
		err := json.Unmarshal([]byte(custom.Config.ValueString()), &tfActionConfig)
		if err != nil {
			diags.AddError("NEAP: Invalid Custom Action", fmt.Sprintf("invalid json config: %s", err.Error()))
			return
		}

		itsiActionConfig := make(map[string]any, len(tfActionConfig))
		for k, v := range tfActionConfig {
			itsiActionConfig[fmt.Sprintf("action.%s.%s", action, k)] = v
		}

		itsiActionJSON, err := json.Marshal(itsiActionConfig)
		if err != nil {
			diags.AddError("NEAP: Invalid Custom Action", fmt.Sprintf("invalid json config: %s", err.Error()))
			return
		}

		config["params"] = string(itsiActionJSON)
	}

	item["config"] = config
	return
}

func NEAPRuleActionsItemFromAPIModel(a map[string]any) (item neapRuleActionsItemModel, diags diag.Diagnostics) {
	item.ExecuteOn = types.StringValue(a["execution_criteria"].(map[string]any)["execute_on"].(string))
	itemType := a["type"].(string)

	config, ok := a["config"].(map[string]any)
	if !ok {
		diags.AddError("NEAP: Invalid Rule Actions", "invalid rule actions item config")
		return
	}

	if itemType == itsiNeapActionNotableEventExecuteAction {
		actionName := config["name"].(string)

		customAction := neapRuleActionsCustomActionModel{
			Type: types.StringValue(actionName),
		}

		var itsiActionParams map[string]any
		err := json.Unmarshal([]byte(config["params"].(string)), &itsiActionParams)
		if err != nil {
			diags.AddError("NEAP: Invalid Custom Action", fmt.Sprintf("invalid json config: %s", err.Error()))
			return
		}

		tfActionParams := make(map[string]any, len(itsiActionParams))
		for k, v := range itsiActionParams {
			param := strings.Join(strings.Split(k, ".")[2:], ".")
			tfActionParams[param] = v
		}

		tfActionParamsJSON, err := json.Marshal(tfActionParams)
		if err != nil {
			diags.AddError("NEAP: Invalid Custom Action", fmt.Sprintf("invalid json config: %s", err.Error()))
			return
		}

		customAction.Config = types.StringValue(string(tfActionParamsJSON))
		item.Custom = []neapRuleActionsCustomActionModel{customAction}
		return
	}

	//item

	var itemField string
	if f, ok := config["field"]; ok {
		itemField = f.(string)
	}

	standardAction := neapStandardActionTypeAndField{itsiNeapActionNotableEventChange, itemField}
	transform := util.ReverseMap(neapStandardActionTfToItsiValueMapping[standardAction])
	v := config["value"].(string)
	if transform != nil {
		if value, ok := transform[v]; ok {
			v = value
		} else {
			diags.AddError("NEAP: Invalid Standard Action", fmt.Sprintf("unsupported value for action type: %s", v))
		}
	}
	(*item.field(standardAction)) = types.StringValue(v)
	return
}

type resourceNEAP struct {
	client models.ClientConfig
}

func NewResourceNEAP() resource.Resource {
	return &resourceNEAP{}
}

func (r *resourceNEAP) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		resp.Diagnostics.AddError("Unable to prepare client", "invalid provider data")
		return
	}
	r.client = client
}

func (r *resourceNEAP) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notable_event_aggregation_policy"
}

func (r *resourceNEAP) criteriaSchema(criteriaType neapCriteriaType) schema.SingleNestedBlock {
	condition := stringdefault.StaticString("OR")
	conflicts := []validator.Object{}
	switch criteriaType {
	case neapCriteriaTypeActivation:
		condition = stringdefault.StaticString("AND")
	case neapCriteriaTypeBreaking:
		conflicts = []validator.Object{objectvalidator.ConflictsWith(path.MatchRelative().AtName("breaking_criteria"))}
	case neapCriteriaTypeFilter:
		conflicts = []validator.Object{
			objectvalidator.ConflictsWith(path.MatchRelative().AtName("pause")),
			objectvalidator.ConflictsWith(path.MatchRelative().AtName("duration")),
			objectvalidator.ConflictsWith(path.MatchRelative().AtName("notable_event_count")),
			objectvalidator.ConflictsWith(path.MatchRelative().AtName("breaking_criteria")),
		}
	default:
		panic("unexpected criteria type")
	}

	return schema.SingleNestedBlock{
		// MarkdownDescription: util.Dedent(`
		// 	An array of data drilldown objects.
		// 	Each data drilldown defines filters for raw data associated with entities that belong to the entity type.
		// `),

		Blocks: map[string]schema.Block{
			"clause": schema.SetNestedBlock{
				// MarkdownDescription: util.Dedent(`
				// 	Further filter down to the raw data associated with the entity
				// 	based on a set of selected entity alias or informational fields.
				// `),
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"notable_event_field": schema.SetNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"field": schema.StringAttribute{
										Required: true,
									},
									"operator": schema.StringAttribute{
										Required: true,
										Validators: []validator.String{
											stringvalidator.OneOf(
												"=",
												"!=",
												">=",
												">",
												"<",
											),
										},
									},
									"value": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: `A wildcard pattern to match against a field value. E.g.: "*"`,
									},
								},
							},
						},
					},
					Attributes: map[string]schema.Attribute{
						"condition": schema.StringAttribute{
							//MarkdownDescription: "Data field.",
							Optional:   true,
							Computed:   true,
							Default:    stringdefault.StaticString("AND"),
							Validators: []validator.String{stringvalidator.OneOf("AND")},
						},
					},
				},
				// Validators: []validator.Set{
				// 	setvalidator.SizeAtLeast(1),
				// },
			},
			"pause": schema.SetNestedBlock{
				MarkdownDescription: "Corresponds to the statement: if the flow of events into the episode paused for %%param.pause%% seconds.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"limit": schema.Int64Attribute{
							Required: true,
						},
					},
				},
				Validators: []validator.Set{setvalidator.SizeAtMost(1)},
			},

			"duration": schema.SetNestedBlock{
				MarkdownDescription: "Corresponds to the statement: if the episode existed for %%param.duration%% seconds.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"limit": schema.Int64Attribute{
							Required: true,
						},
					},
				},
				Validators: []validator.Set{setvalidator.SizeAtMost(1)},
			},

			"notable_event_count": schema.SetNestedBlock{
				MarkdownDescription: "Corresponds to the statement: if the number of events in this episode is %%operator%% %%limit%%.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"operator": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("==", "!=", ">=", "<=", ">", "<"),
							},
						},
						"limit": schema.Int64Attribute{
							Required: true,
						},
					},
				},
				Validators: []validator.Set{setvalidator.SizeAtMost(1)},
			},
			"breaking_criteria": schema.SetNestedBlock{
				MarkdownDescription: "Corresponds to the statement: if the episode is broken. Note: applicable only for the Activation Criteria.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"config": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
						},
					},
				},
				Validators: []validator.Set{setvalidator.SizeAtMost(1)},
			},
		},
		Attributes: map[string]schema.Attribute{
			"condition": schema.StringAttribute{
				MarkdownDescription: "Computed depends of the criteria type. In case of activation_criteria condition equals AND, otherwise - OR.",
				//Optional:            true,
				Computed: true,
				Default:  condition,
				// Validators: []validator.String{stringvalidator.OneOf("AND")},
			},
		},
		Validators: append([]validator.Object{objectvalidator.IsRequired()}, conflicts...),
	}
}

func (r *resourceNEAP) ruleActionsSchema() schema.SetNestedBlock {
	itemTypes := []string{
		"custom",
		//standard actions:
		"change_severity",
		"change_status",
		"change_owner",
		"comment",
	}

	itemTypePaths := make([]path.Expression, len(itemTypes))
	for i, itemType := range itemTypes {
		itemTypePaths[i] = path.MatchRelative().AtParent().AtName(itemType)
	}

	return schema.SetNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"item": schema.SetNestedBlock{
					NestedObject: schema.NestedBlockObject{
						Blocks: map[string]schema.Block{
							"custom": schema.SetNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
										},
										"config": schema.StringAttribute{
											Optional: true,
											Computed: true,
											Default:  stringdefault.StaticString("{}"),
										},
									},
								},
								Validators: []validator.Set{setvalidator.SizeAtMost(1)},
							},
						},
						Attributes: map[string]schema.Attribute{

							"execute_on": schema.StringAttribute{
								MarkdownDescription: `ExecutionCriteria is essentially the criteria answering: "on which events is ActionItem applicable".`,
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("GROUP"),
								Validators: []validator.String{
									stringvalidator.OneOf("GROUP", "ALL", "FILTER", "THIS"),
								},
							},

							// standard NEAP actions

							neapActionChangeSeverity: schema.StringAttribute{
								MarkdownDescription: "Change the severity of the episode to the specified value.",
								Optional:            true,
								//Computed:            true,
								//Default:             stringdefault.StaticString(""),
								Validators: []validator.String{
									stringvalidator.OneOf(*util.GetSupportedSeverities()...),
									stringvalidator.ExactlyOneOf(itemTypePaths...),
								},
							},

							neapActionChangeStatus: schema.StringAttribute{
								MarkdownDescription: "Change the status of the episode to the specified value.",
								Optional:            true,
								//Computed:            true,
								//Default:             stringdefault.StaticString(""),
								Validators: []validator.String{stringvalidator.OneOf(*util.GetSupportedStatuses()...)},
							},

							neapActionChangeOwner: schema.StringAttribute{
								MarkdownDescription: "Change the owner of the episode to the specified value.",
								Optional:            true,
							},

							neapActionComment: schema.StringAttribute{
								MarkdownDescription: "Add a comment to the episode.",
								Optional:            true,
							},

							// "type": schema.StringAttribute{
							// 	Required: true,
							// },
							// "config": schema.StringAttribute{
							// 	Optional: true,
							// 	Computed: true,
							// 	Default:  stringdefault.StaticString("{}"),
							// },

						},
						Validators: []validator.Object{
							//objectvalidator.ExactlyOneOf()
							//objectvalidator.ExactlyOneOf(itemTypePaths...), //!!!!!!!!!
						},
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"condition": schema.StringAttribute{
					Computed: true,
					Default:  stringdefault.StaticString("AND"),
				},
			},
		},
	}

}

func (r *resourceNEAP) ruleSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{

		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"activation_criteria": r.criteriaSchema(neapCriteriaTypeActivation),
				"actions":             r.ruleActionsSchema(),
			},
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "ID of the notable event aggregation policy rule.",
					Computed:            true,
					//Optional:            true,
					//PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "The description of the notable event aggregation policy rule.",
					Optional:            true,
					Computed:            true,
					Default:             stringdefault.StaticString(""),
				},
				"title": schema.StringAttribute{
					MarkdownDescription: "The title of the notable event aggregation policy rule.",
					Optional:            true,
					Computed:            true,
					Default:             stringdefault.StaticString(""),
				},
				"priority": schema.Int64Attribute{
					MarkdownDescription: "The priority of the notable event aggregation policy rule.",
					Optional:            true,
					Computed:            true,
					Default:             int64default.StaticInt64(5),
					Validators:          []validator.Int64{int64validator.Between(0, 10)},
				},
			},
		},
	}

}

func (r *resourceNEAP) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Notable Event Aggregation Policy object within ITSI.",
		Blocks: map[string]schema.Block{
			"breaking_criteria": r.criteriaSchema(neapCriteriaTypeBreaking),
			"filter_criteria":   r.criteriaSchema(neapCriteriaTypeFilter),
			"rule":              r.ruleSchema(),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the NEAP.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the notable event aggregation policy.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the notable event aggregation policy.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the notable event aggregation policy is disabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"priority": schema.Int64Attribute{
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(5),
				Validators: []validator.Int64{int64validator.Between(0, 10)},
			},
			"split_by_field": schema.SetAttribute{
				MarkdownDescription: "Fields to split an episode by.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},

			"entity_factor_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the entity factor is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"run_time_based_actions_once": schema.BoolAttribute{
				MarkdownDescription: util.Dedent(`
					If you create an action to add a comment after an episode has existed for 60 seconds, a comment will only be added once for the episode.
					There are 2 conditions that trigger time-based actions:
					- The episode existed for (duration)
					- The flow of events into the episode paused for (pause)
				`),
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"service_topology_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the service topology is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},

			"group_title": schema.StringAttribute{
				MarkdownDescription: "The default title of each episode created by the notable event aggregation policy. (Episode Title)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%title%"),
			},
			"group_description": schema.StringAttribute{
				MarkdownDescription: "The description of each episode created by the notable event aggregation policy. (Episode Description)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%description%"),
			},
			"group_severity": schema.StringAttribute{
				MarkdownDescription: "The default severity of each episode created by the notable event aggregation policy. (Episode Severity)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%severity%"),
				Validators: []validator.String{stringvalidator.OneOf(
					append(
						*util.GetSupportedSeverities(),
						[]string{
							"%severity%",
							"%last_severity%",
							"%lowest_severity%",
							"%highest_severity%"}...)...),
				},
			},
			"group_assignee": schema.StringAttribute{
				MarkdownDescription: "The default owner of each episode created by the notable event aggregation policy. (Episode Asignee)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%owner%"),
			},
			"group_status": schema.StringAttribute{
				MarkdownDescription: "The default status of each episode created by the notable event aggregation policy.  (Episode Status)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%status%"),

				Validators: []validator.String{stringvalidator.OneOf(
					append(
						*util.GetSupportedStatuses(),
						[]string{
							"%status%",
							"%last_status%"}...)...),
				},
			},
			"group_instruction": schema.StringAttribute{
				MarkdownDescription: "The default instructions of each episode created by the notable event aggregation policy.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("%itsi_instruction%"),
				Validators: []validator.String{stringvalidator.OneOf(
					"%itsi_instruction%",
					"%last_instruction%",
					"%all_instruction%",
					"%custom_instruction%",
				)},
			},
			"group_custom_instruction": schema.StringAttribute{
				MarkdownDescription: "The custom instruction of each episode created by the notable event aggregation policy.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"group_dashboard": schema.StringAttribute{
				MarkdownDescription: "Customize the Episode dashboard using a JSON-formatted dashboard definition. The first notable event's fields are available to use as tokens in the dashboard.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"group_dashboard_context": schema.StringAttribute{
				MarkdownDescription: "Dashboard Tokens",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("first"),
				Validators:          []validator.String{stringvalidator.OneOf("first", "last")},
			},
		},
	}
}

func (r *resourceNEAP) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state neapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := notableEventAggregationPolicyBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read NEAP", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &neapModel{})...)
		return
	}

	state, diags := newAPIParser[neapModel](b, new(neapParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceNEAP) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan neapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	//plan.normalize()

	base, diags := newAPIBuilder[neapModel](r.client, new(neapBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create NEAP", err.Error())
		return
	}

	state := plan
	state, diags = newAPIParser[neapModel](base, new(neapParseWorkflow)).parse(ctx, base)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	//state.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceNEAP) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan neapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	//plan.normalize()

	base, diags := newAPIBuilder[neapModel](r.client, new(neapBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update NEAP", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update NEAP", "NEAP not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update NEAP", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceNEAP) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state neapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	base := notableEventAggregationPolicyBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete NEAP", err.Error())
		return
	}
	if b == nil {
		return
	}
	if err := b.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to delete NEAP", err.Error())
		return
	}
}

func (r *resourceNEAP) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	b := notableEventAggregationPolicyBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find NEAP", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("NEAP not found", fmt.Sprintf("NEAP '%s' not found", req.ID))
		return
	}

	state, diags := newAPIParser[neapModel](b, new(neapParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
