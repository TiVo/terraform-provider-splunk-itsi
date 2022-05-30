package provider

import "regexp"

var TestServiceResourceValidationDataProvider = []struct {
	config      string
	expectError *regexp.Regexp
}{
	{
		config: `
            resource "test_service_resource" "test_service_resource_validation" {
                title    = "TEST KPI VALIDATION URGENCY LESS THAN"
                description = "Terraform unit test"
                kpi {
                   title = "test kpi urgency less"
                   threshold_template_id = "0000001_tt"
                   base_search_id = "0000001_bs"
                   base_search_metric =  "0000001_bsm"
                   urgency = -1
                }
            }
        `,
		expectError: regexp.MustCompile(`.*kpi\.0\.urgency.*`),
	},
	{
		config: `
            resource "test_service_resource" "test_service_resource_validation" {
                title    = "TEST KPI VALIDATION URGENCY MORE THAN"
                description = "Terraform unit test"
                kpi {
                   title = "test kpi urgency less"
                   threshold_template_id = "0000000_tt"
                   base_search_id = "0000000_bs"
                   base_search_metric =  "0000000_bsm"
                   urgency = 12
                }
            }
        `,
		expectError: regexp.MustCompile(`.*kpi\.0\.urgency.*`),
	},
	{
		config: `
            resource "test_service_resource" "test_service_resource_validation" {
                title    = "TEST KPI VALIDATION TYPE"
                description = "Terraform unit test"
                kpi {
                   title = "test kpi type"
                   threshold_template_id = "0000002_tt"
                   base_search_id = "0000002_bs"
                   base_search_metric =  "0000002_bsm"
                   urgency = 8
                   type = "random_string"
                }
            }
        `,
		expectError: regexp.MustCompile(`.*kpi\.0\.type.*`),
	},
	{
		config: `
             resource "test_service_resource" "test_service_resource_validation" {
                 title    = "TEST ENTITY RULE VALIDATION RULE"
                 description = "Terraform unit test"
                 entity_rules {
                    rule {
                        field      = "entityTitle"
                        field_type = "random_string"
                        rule_type  = "matches"
                        value      = "android_streamer"
                    }
                 }
             }
        `,
		expectError: regexp.MustCompile(`.*entity_rules\.0\.rule\.0\.field_type.*`),
	},
	{
		config: `
             resource "test_service_resource" "test_service_resource_validation" {
                 title    = "TEST ENTITY RULE VALIDATION FIELD"
                 description = "Terraform unit test"
                 entity_rules {
                    rule {
                        field      = "entityTitle"
                        rule_type  = "random_string"
                        field_type = "alias"
                        value      = "android_streamer"
                    }
                 }
             }
        `,
		expectError: regexp.MustCompile(`.*entity_rules\.0\.rule\.0\.rule_type.*`),
	},
	{
		config: `
            resource "test_service_resource" "test_service_resource_validation" {
                title    = "TEST KPI wrong type"
                description = "Terraform unit test"
                kpi {
					title = "test kpi urgency less"
					threshold_template_id = "0000001_tt"
					base_search_id = "0000001_bs"
					base_search_metric =  "0000001_bsm"
					urgency = 8
					type = "random_string"
                }
            }
        `,
		expectError: regexp.MustCompile(`.*kpi\.0\.type to be one of \[kpis_primary service_health\].*`),
	},
}

var TestServiceResourceCreateDataProvider = []struct {
	// test case description
	description                string
	resourceName               string
	config                     string
	inputBaseSearchId          string
	inputBaseSearch            string
	inputThresholdTemplateId   string
	inputThresholdTemplate     string
	expectedServicePostBody    string
	expectedApiCallsCount      int
	expectedResourceAttributes map[string]string
	serviceIdToSet             string
}{
	{
		description:  "No optional fields provided",
		resourceName: "service_create_test",
		config: `
	        resource "test_service_resource" "service_create_test" {
	            title    = "TEST"
	            description = "Terraform unit test"
	        }
	    `,
		inputBaseSearchId:        "",
		inputBaseSearch:          "",
		inputThresholdTemplateId: "",
		inputThresholdTemplate:   "",
		expectedServicePostBody: `
	        {
	           "description":"Terraform unit test",
	           "enabled":0,
	           "entity_rules":[],
	           "kpis":[],
	           "object_type":"service",
	           "sec_grp":"default_itsi_security_group",
	           "services_depends_on":[],
	           "title":"TEST"
	        }
	        `,
		// [1] from service create: post the new service
		// [2] from service create: read from service to populate resource data
		// [3] from service delete: performs service delete
		expectedApiCallsCount: 3,
		expectedResourceAttributes: map[string]string{
			"title":          "TEST",
			"description":    "Terraform unit test",
			"id":             "service_create_no_option",
			"security_group": "default_itsi_security_group",
			"enabled":        "false",
		},
		serviceIdToSet: "service_create_no_option",
	},
	{
		description:  "services_depends_on field population",
		resourceName: "service_create_test",
		config: `
        resource "test_service_resource" "service_create_test" {
            title    = "TEST"
            description = "Terraform unit test"
            service_depends_on {
                kpis = ["SHKPI-00000000"]
                service = "00000000"
            }
            service_depends_on {
                kpis = ["SHKPI-00000001","0000-1234"]
                service = "00000001"
            }
        }
        `,
		inputBaseSearchId:        "",
		inputBaseSearch:          "",
		inputThresholdTemplateId: "",
		inputThresholdTemplate:   "",
		expectedServicePostBody: `{
            "description":"Terraform unit test",
            "enabled":0,
            "entity_rules":[],
            "kpis":[],
            "object_type":"service",
            "sec_grp":"default_itsi_security_group",
            "services_depends_on":[
               {
                  "kpis_depending_on":[
                     "SHKPI-00000000"
                  ],
                  "serviceid":"00000000"
               },
               {
                  "kpis_depending_on":[
                     "SHKPI-00000001",
                     "0000-1234"
                  ],
                  "serviceid":"00000001"
               }
            ],
            "title":"TEST"
         }`,
		expectedApiCallsCount: 3,
		expectedResourceAttributes: map[string]string{
			//.# length of set
			//.% length of map
			"service_depends_on.0.%": "2",
			//len(service_depends_on[0][kpis]) == 2
			"service_depends_on.0.kpis.#":  "2",
			"service_depends_on.0.kpis.0":  "0000-1234",
			"service_depends_on.0.kpis.1":  "SHKPI-00000001",
			"service_depends_on.0.service": "00000001",
			//len(service_depends_on[1][kpis]) == 1
			"service_depends_on.1.kpis.#":  "1",
			"service_depends_on.1.kpis.0":  "SHKPI-00000000",
			"service_depends_on.1.service": "00000000",
		},
		serviceIdToSet: "service_create_test_service_depends_on",
	},
	{
		description:  "service tags field population",
		resourceName: "service_create_test",
		config: `
            resource "test_service_resource" "service_create_test" {
                title    = "TEST"
                description = "Terraform unit test"
                tags = ["tag1", "tag2", "tag3"]
            }
        `,
		inputBaseSearchId:        "",
		inputBaseSearch:          "",
		inputThresholdTemplateId: "",
		inputThresholdTemplate:   "",
		expectedServicePostBody: `
            {
               "description":"Terraform unit test",
               "enabled":0,
               "entity_rules":[],
               "kpis":[],
               "object_type":"service",
               "sec_grp":"default_itsi_security_group",
               "service_tags":{
                  "tags":[
                     "tag3",
                     "tag2",
                     "tag1"
                  ]
               },
               "services_depends_on":[],
               "title":"TEST"
            }
            `,
		expectedApiCallsCount: 3,
		expectedResourceAttributes: map[string]string{
			"tags.#": "3",
			"tags.0": "tag1",
			"tags.1": "tag2",
			"tags.2": "tag3",
		},
		serviceIdToSet: "service_create_test_tags",
	},
	{
		description:  "entity filter field population",
		resourceName: "service_create_test",
		config: `
            resource "test_service_resource" "service_create_test" {
                title    = "TEST"
                description = "Terraform unit test"
                entity_rules {
                    rule {
                        field      = "entityTitle"
                        field_type = "alias"
                        rule_type  = "matches"
                        value      = "android_streamer"
                    }
                    rule {
                        field      = "entityField"
                        field_type = "info"
                        rule_type  = "not"
                        value      = "android_mobile"
                    }
                }
            }
        `,
		inputBaseSearchId:        "",
		inputBaseSearch:          "",
		inputThresholdTemplateId: "",
		inputThresholdTemplate:   "",
		expectedServicePostBody: `
        {
           "description":"Terraform unit test",
           "enabled":0,
           "entity_rules":[
              {
                 "rule_condition":"AND",
                 "rule_items":[
                    {
                       "field":"entityField",
                       "field_type":"info",
                       "rule_type":"not",
                       "value":"android_mobile"
                    },
                    {
                       "field":"entityTitle",
                       "field_type":"alias",
                       "rule_type":"matches",
                       "value":"android_streamer"
                    }
                 ]
              }
           ],
           "kpis":[],
           "object_type":"service",
           "sec_grp":"default_itsi_security_group",
           "services_depends_on":[],
           "title":"TEST"
        }
        `,
		expectedApiCallsCount: 3,
		expectedResourceAttributes: map[string]string{
			"entity_rules.0.rule.#":            "2",
			"entity_rules.0.rule.0.field":      "entityField",
			"entity_rules.0.rule.0.field_type": "info",
			"entity_rules.0.rule.0.rule_type":  "not",
			"entity_rules.0.rule.0.value":      "android_mobile",
			"entity_rules.0.rule.1.field":      "entityTitle",
			"entity_rules.0.rule.1.field_type": "alias",
			"entity_rules.0.rule.1.rule_type":  "matches",
			"entity_rules.0.rule.1.value":      "android_streamer",
		},
		serviceIdToSet: "service_create_entity_rules",
	},
	{
		description:  "kpi population",
		resourceName: "service_create_test",
		config: `
            resource "test_service_resource" "service_create_test" {
                title    = "TEST"
                description = "Terraform unit test"
                kpi {
                    title = "test kpi 1"
                    base_search_id = "1234_bs"
                    base_search_metric = "UT: Forwarder Count"
                    threshold_template_id="1234_tt"
                }
                kpi {
                    title = "test kpi 2"
                    base_search_id = "1234_bs"
                    base_search_metric = "UT: MAX Forwarder Count"
                    threshold_template_id="1234_tt"
                    urgency = 11
                }
            }
        `,
		inputBaseSearchId: "1234_bs",
		inputBaseSearch: `
        {
          "actions": "",
          "alert_lag": "31",
          "alert_period": "5",
          "base_search": "testSearch",
          "description": "test description",
          "entity_alias_filtering_fields": "test_alias",
          "entity_breakdown_id_fields": "host",
          "entity_id_fields": "host",
          "is_entity_breakdown": false,
          "is_service_entity_filter": false,
          "metric_qualifier": "",
          "metrics": [
            {
              "_key": "1234_bsm",
              "aggregate_statop": "ut_dc",
              "entity_statop": "ut_avg",
              "fill_gaps": "null_value",
              "gap_custom_alert_value": "0",
              "gap_severity": "unknown",
              "gap_severity_color": "#555555",
              "gap_severity_color_light": "#000000",
              "gap_severity_value": "-1",
              "threshold_field": "test_host",
              "title": "UT: Forwarder Count",
              "unit": ""
            },
            {
              "_key": "5678_bsm",
              "aggregate_statop": "ut_p90",
              "entity_statop": "ut_max",
              "fill_gaps": "null_value",
              "gap_custom_alert_value": "0",
              "gap_severity": "unknown",
              "gap_severity_color": "#555555",
              "gap_severity_color_light": "#000000",
              "gap_severity_value": "-1",
              "threshold_field": "test_host",
              "title": "UT: MAX Forwarder Count",
              "unit": "test unit"
            }
          ],
          "objectType": "kpi_base_search",
          "search_alert_earliest": "5",
          "sec_grp": "default_itsi_security_group",
          "title": "test title metric",
          "object_type": "kpi_base_search",
          "_key": "1234_bs"
        }
        `,
		inputThresholdTemplateId: "1234_tt",
		inputThresholdTemplate: `
        {
           "adaptive_thresholding_training_window":"-30d",
           "adaptive_thresholds_is_enabled":true,
           "description":"kpi_threshold_template",
           "objectType":"kpi_threshold_template",
           "sec_grp":"default_itsi_security_group",
           "time_variate_thresholds":true,
           "time_variate_thresholds_specification":{
              "policies":{
                 "default_policy":{
                    "aggregate_thresholds":{
                       "baseSeverityColor":"#0000000",
                       "baseSeverityColorLight":"#55555",
                       "baseSeverityLabel":"critical",
                       "baseSeverityValue":6,
                       "gaugeMax":4,
                       "gaugeMin":-4,
                       "isMaxStatic":false,
                       "isMinStatic":false,
                       "metricField":"",
                       "renderBoundaryMax":4,
                       "renderBoundaryMin":-4,
                       "search":"",
                       "thresholdLevels":[
                          {
                             "dynamicParam":-2.75,
                             "severityColor":"#F26A35",
                             "severityColorLight":"#FBCBB9",
                             "severityLabel":"high",
                             "severityValue":5,
                             "thresholdValue":-2.75
                          },
                          {
                             "dynamicParam":1.25,
                             "severityColor":"#FFE98C",
                             "severityColorLight":"#FFF4C5",
                             "severityLabel":"low",
                             "severityValue":3,
                             "thresholdValue":1.25
                          },
                          {
                             "dynamicParam":-2.5,
                             "severityColor":"#FCB64E",
                             "severityColorLight":"#FEE6C1",
                             "severityLabel":"medium",
                             "severityValue":4,
                             "thresholdValue":-2.5
                          },
                          {
                             "dynamicParam":-1.75,
                             "severityColor":"#FFE98C",
                             "severityColorLight":"#FFF4C5",
                             "severityLabel":"low",
                             "severityValue":3,
                             "thresholdValue":-1.75
                          },
                          {
                             "dynamicParam":-1.25,
                             "severityColor":"#99D18B",
                             "severityColorLight":"#DCEFD7",
                             "severityLabel":"normal",
                             "severityValue":2,
                             "thresholdValue":-1.25
                          },
                          {
                             "dynamicParam":2.25,
                             "severityColor":"#F26A35",
                             "severityColorLight":"#FBCBB9",
                             "severityLabel":"high",
                             "severityValue":5,
                             "thresholdValue":2.25
                          },
                          {
                             "dynamicParam":1.75,
                             "severityColor":"#FCB64E",
                             "severityColorLight":"#FEE6C1",
                             "severityLabel":"medium",
                             "severityValue":4,
                             "thresholdValue":1.75
                          }
                       ]
                    },
                    "entity_thresholds":{
                       "baseSeverityColor":"#AED3E5",
                       "baseSeverityColorLight":"#E3F0F6",
                       "baseSeverityLabel":"info",
                       "baseSeverityValue":1,
                       "gaugeMax":0,
                       "gaugeMin":0,
                       "isMaxStatic":false,
                       "isMinStatic":false,
                       "metricField":"",
                       "renderBoundaryMax":0,
                       "renderBoundaryMin":0,
                       "search":"",
                       "thresholdLevels":[]
                    },
                    "policy_type":"stdev",
                    "time_blocks":[],
                    "title":"default_policy"
                 }
              }
           },
           "title":"TEST stdev, not windowed, both bad, 1.25 strictness, 0.50 cascade, -30d adaptive window",
           "object_type":"kpi_threshold_template",
           "_key":"1234_tt"
        }
        `,
		expectedServicePostBody: `
       {
          "description":"Terraform unit test",
          "enabled":0,
          "entity_rules":[
          ],
          "kpis":[
             {
                "_key":"5fd830adadebc708df764238712808cbc99e9e40",
                "adaptive_thresholding_training_window":"-30d",
                "adaptive_thresholds_is_enabled":true,
                "aggregate_statop":"ut_p90",
                "aggregate_thresholds":null,
                "alert_lag":"31",
                "alert_period":"5",
                "base_search":"testSearch",
                "base_search_id":"1234_bs",
                "base_search_metric":"5678_bsm",
                "entity_breakdown_id_fields":"host",
                "entity_id_fields":"host",
                "entity_statop":"ut_max",
                "entity_thresholds":null,
                "fill_gaps":"null_value",
                "gap_custom_alert_value":"0",
                "gap_severity":"unknown",
                "gap_severity_color":"#555555",
                "gap_severity_color_light":"#000000",
                "gap_severity_value":"-1",
                "is_entity_breakdown":false,
                "is_service_entity_filter":false,
                "kpi_threshold_template_id":"1234_tt",
                "search_alert_earliest":"5",
                "search_type":"shared_base",
                "threshold_field":"test_host",
                "time_variate_thresholds":true,
                "time_variate_thresholds_specification":{
                   "policies":{
                      "default_policy":{
                         "aggregate_thresholds":{
                            "baseSeverityColor":"#0000000",
                            "baseSeverityColorLight":"#55555",
                            "baseSeverityLabel":"critical",
                            "baseSeverityValue":6,
                            "gaugeMax":4,
                            "gaugeMin":-4,
                            "isMaxStatic":false,
                            "isMinStatic":false,
                            "metricField":"",
                            "renderBoundaryMax":4,
                            "renderBoundaryMin":-4,
                            "search":"",
                            "thresholdLevels":[
                               {
                                  "dynamicParam":-2.75,
                                  "severityColor":"#F26A35",
                                  "severityColorLight":"#FBCBB9",
                                  "severityLabel":"high",
                                  "severityValue":5,
                                  "thresholdValue":-2.75
                               },
                               {
                                  "dynamicParam":1.25,
                                  "severityColor":"#FFE98C",
                                  "severityColorLight":"#FFF4C5",
                                  "severityLabel":"low",
                                  "severityValue":3,
                                  "thresholdValue":1.25
                               },
                               {
                                  "dynamicParam":-2.5,
                                  "severityColor":"#FCB64E",
                                  "severityColorLight":"#FEE6C1",
                                  "severityLabel":"medium",
                                  "severityValue":4,
                                  "thresholdValue":-2.5
                               },
                               {
                                  "dynamicParam":-1.75,
                                  "severityColor":"#FFE98C",
                                  "severityColorLight":"#FFF4C5",
                                  "severityLabel":"low",
                                  "severityValue":3,
                                  "thresholdValue":-1.75
                               },
                               {
                                  "dynamicParam":-1.25,
                                  "severityColor":"#99D18B",
                                  "severityColorLight":"#DCEFD7",
                                  "severityLabel":"normal",
                                  "severityValue":2,
                                  "thresholdValue":-1.25
                               },
                               {
                                  "dynamicParam":2.25,
                                  "severityColor":"#F26A35",
                                  "severityColorLight":"#FBCBB9",
                                  "severityLabel":"high",
                                  "severityValue":5,
                                  "thresholdValue":2.25
                               },
                               {
                                  "dynamicParam":1.75,
                                  "severityColor":"#FCB64E",
                                  "severityColorLight":"#FEE6C1",
                                  "severityLabel":"medium",
                                  "severityValue":4,
                                  "thresholdValue":1.75
                               }
                            ]
                         },
                         "entity_thresholds":{
                            "baseSeverityColor":"#AED3E5",
                            "baseSeverityColorLight":"#E3F0F6",
                            "baseSeverityLabel":"info",
                            "baseSeverityValue":1,
                            "gaugeMax":0,
                            "gaugeMin":0,
                            "isMaxStatic":false,
                            "isMinStatic":false,
                            "metricField":"",
                            "renderBoundaryMax":0,
                            "renderBoundaryMin":0,
                            "search":"",
                            "thresholdLevels":[
                            ]
                         },
                         "policy_type":"stdev",
                         "time_blocks":[
                         ],
                         "title":"default_policy"
                      }
                   }
                },
                "title":"test kpi 2",
                "type":"kpis_primary",
                "unit":"test unit",
                "urgency":11
             },
             {
                "_key":"72eeeddcd9dc12d8010e54ff4e882a0028a94498",
                "adaptive_thresholding_training_window":"-30d",
                "adaptive_thresholds_is_enabled":true,
                "aggregate_statop":"ut_dc",
                "aggregate_thresholds":null,
                "alert_lag":"31",
                "alert_period":"5",
                "base_search":"testSearch",
                "base_search_id":"1234_bs",
                "base_search_metric":"1234_bsm",
                "entity_breakdown_id_fields":"host",
                "entity_id_fields":"host",
                "entity_statop":"ut_avg",
                "entity_thresholds":null,
                "fill_gaps":"null_value",
                "gap_custom_alert_value":"0",
                "gap_severity":"unknown",
                "gap_severity_color":"#555555",
                "gap_severity_color_light":"#000000",
                "gap_severity_value":"-1",
                "is_entity_breakdown":false,
                "is_service_entity_filter":false,
                "kpi_threshold_template_id":"1234_tt",
                "search_alert_earliest":"5",
                "search_type":"shared_base",
                "threshold_field":"test_host",
                "time_variate_thresholds":true,
                "time_variate_thresholds_specification":{
                   "policies":{
                      "default_policy":{
                         "aggregate_thresholds":{
                            "baseSeverityColor":"#0000000",
                            "baseSeverityColorLight":"#55555",
                            "baseSeverityLabel":"critical",
                            "baseSeverityValue":6,
                            "gaugeMax":4,
                            "gaugeMin":-4,
                            "isMaxStatic":false,
                            "isMinStatic":false,
                            "metricField":"",
                            "renderBoundaryMax":4,
                            "renderBoundaryMin":-4,
                            "search":"",
                            "thresholdLevels":[
                               {
                                  "dynamicParam":-2.75,
                                  "severityColor":"#F26A35",
                                  "severityColorLight":"#FBCBB9",
                                  "severityLabel":"high",
                                  "severityValue":5,
                                  "thresholdValue":-2.75
                               },
                               {
                                  "dynamicParam":1.25,
                                  "severityColor":"#FFE98C",
                                  "severityColorLight":"#FFF4C5",
                                  "severityLabel":"low",
                                  "severityValue":3,
                                  "thresholdValue":1.25
                               },
                               {
                                  "dynamicParam":-2.5,
                                  "severityColor":"#FCB64E",
                                  "severityColorLight":"#FEE6C1",
                                  "severityLabel":"medium",
                                  "severityValue":4,
                                  "thresholdValue":-2.5
                               },
                               {
                                  "dynamicParam":-1.75,
                                  "severityColor":"#FFE98C",
                                  "severityColorLight":"#FFF4C5",
                                  "severityLabel":"low",
                                  "severityValue":3,
                                  "thresholdValue":-1.75
                               },
                               {
                                  "dynamicParam":-1.25,
                                  "severityColor":"#99D18B",
                                  "severityColorLight":"#DCEFD7",
                                  "severityLabel":"normal",
                                  "severityValue":2,
                                  "thresholdValue":-1.25
                               },
                               {
                                  "dynamicParam":2.25,
                                  "severityColor":"#F26A35",
                                  "severityColorLight":"#FBCBB9",
                                  "severityLabel":"high",
                                  "severityValue":5,
                                  "thresholdValue":2.25
                               },
                               {
                                  "dynamicParam":1.75,
                                  "severityColor":"#FCB64E",
                                  "severityColorLight":"#FEE6C1",
                                  "severityLabel":"medium",
                                  "severityValue":4,
                                  "thresholdValue":1.75
                               }
                            ]
                         },
                         "entity_thresholds":{
                            "baseSeverityColor":"#AED3E5",
                            "baseSeverityColorLight":"#E3F0F6",
                            "baseSeverityLabel":"info",
                            "baseSeverityValue":1,
                            "gaugeMax":0,
                            "gaugeMin":0,
                            "isMaxStatic":false,
                            "isMinStatic":false,
                            "metricField":"",
                            "renderBoundaryMax":0,
                            "renderBoundaryMin":0,
                            "search":"",
                            "thresholdLevels":[
                            ]
                         },
                         "policy_type":"stdev",
                         "time_blocks":[],
                         "title":"default_policy"
                      }
                   }
                },
                "title":"test kpi 1",
                "type":"kpis_primary",
                "unit":"",
                "urgency":5
             }
          ],
          "object_type":"service",
          "sec_grp":"default_itsi_security_group",
          "services_depends_on":[],
          "title":"TEST"
       }
       `,
		// [1] from service create: to populate base search dependency
		// [2] from service create: to populate threshold template dependency
		// [3] from service create: post the new service
		// [4] from service create: read from service to populate resource data
		// [5] from service delete: performs service delete
		expectedApiCallsCount:      5,
		expectedResourceAttributes: map[string]string{},
		serviceIdToSet:             "service_test_create_kpis",
	},
}
