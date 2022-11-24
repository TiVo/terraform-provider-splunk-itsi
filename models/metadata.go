package models

const (
	metadataConfig = `
backup_restore:
    rest_interface: backup_restore_interface
    object_type: backup_restore
    rest_key_field: _key
    tfid_field: _key

team:
    rest_interface: itoa_interface
    object_type: team
    rest_key_field: _key
    tfid_field: title

entity:
    rest_interface: itoa_interface
    object_type: entity
    rest_key_field: _key
    tfid_field: title
    max_page_size: 1000
    generate_key: true

entity_type:
    rest_interface: itoa_interface
    object_type: entity_type
    rest_key_field: _key
    tfid_field: title

service:
    rest_interface: itoa_interface
    object_type: service
    rest_key_field: _key
    tfid_field: title
    max_page_size: 100
    generate_key: true

base_service_template:
    rest_interface: itoa_interface
    object_type: base_service_template
    rest_key_field: _key
    tfid_field: title

kpi_base_search:
    rest_interface: itoa_interface
    object_type: kpi_base_search
    rest_key_field: _key
    tfid_field: title
    generate_key: true

deep_dive:
    rest_interface: itoa_interface
    object_type: deep_dive
    rest_key_field: _key
    tfid_field: title

glass_table:
    object_type: glass_table
    rest_key_field: _key
    tfid_field: _key
    rest_interface: itoa_interface

home_view:
    rest_interface: itoa_interface
    object_type: home_view
    rest_key_field: _key
    tfid_field: title

kpi_template:
    rest_interface: itoa_interface
    object_type: kpi_template
    rest_key_field: _key
    tfid_field: title

kpi_threshold_template:
    rest_interface: itoa_interface
    object_type: kpi_threshold_template
    rest_key_field: _key
    tfid_field: title
    max_page_size: 100
    generate_key: true

event_management_state:
    rest_interface: itoa_interface
    object_type: event_management_state
    rest_key_field: _key
    tfid_field: title

entity_relationship:
    rest_interface: itoa_interface
    object_type: entity
    rest_key_field: _key
    tfid_field: _key
    max_page_size: 1000


entity_relationship_rule:
    rest_interface: itoa_interface
    object_type: entity
    rest_key_field: _key
    tfid_field: _key
    max_page_size: 1000


notable_event_aggregation_policy:
    rest_interface: event_management_interface
    object_type: notable_event_aggregation_policy
    rest_key_field: _key
    tfid_field: title

correlation_search:
    rest_interface: event_management_interface
    object_type: correlation_search
    rest_key_field: name
    tfid_field: name

notable_event_group:
    rest_interface: event_management_interface
    object_type: notable_event_group
    rest_key_field: _key
    tfid_field: _key

# notable_event_comment does not support bulk get
#     notable_event_comment:
#        rest_interface: event_management_interface
#        object_type:  notable_event_comment
#        rest_key_field:  _key
#        tfid_field:   title
#
`
)
