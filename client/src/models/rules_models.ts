export interface RuleCondition {
  id?: number | null;
  rule_id?: number | null;
  parent_id?: number | null;
  is_group: boolean;
  match_type: string;
  field: string;
  operator: string;
  value: string;
  position: number;
}

export interface RuleAction {
  id?: number | null;
  rule_id?: number | null;
  action_type: string;
  value: string;
  position: number;
}

export interface Rule {
  id?: number | null;
  user_id?: number | null;
  name: string;
  is_active: boolean;
  match_type: string;
  effective_date: string | null;
  conditions: RuleCondition[];
  actions: RuleAction[];
  created_at?: string;
  updated_at?: string;
}

export interface RuleConditionReq {
  is_group: boolean;
  match_type: string;
  field: string;
  operator: string;
  value: string;
  conditions: RuleConditionReq[];
}
