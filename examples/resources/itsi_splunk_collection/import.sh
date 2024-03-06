terraform import itsi_splunk_collection.example {{owner}}/{{app}}/{{name}}
#OR
terraform import itsi_service.example {{app}}/{{name}}
#OR
terraform import itsi_service.example {{name}}

# NOTE:
# When the collection's owner and/or app are not provided, the default values of "nobody" and "itsi" are assumed respectively.
