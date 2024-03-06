terraform import itsi_collection_data.example {{owner}}/{{app}}/{{name}}:{{scope}}
OR
terraform import itsi_collection_data.example {{app}}/{{name}}:{{scope}}
#OR
terraform import itsi_collection_data.example {{name}}:{{scope}}
#OR
terraform import itsi_collection_data.example {{name}}

# NOTE:
# When the collection's owner and/or app are not provided, the default values of "nobody" and "itsi" are assumed respectively.
# When scope is ommited, the "default" scope is assumed.
