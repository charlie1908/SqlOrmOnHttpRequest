package Core

import shared "httpRequestName/Shared"

// Mapping and index name. Elasticsearch index doctypes now deprecated
const (
	loginMappings = ` 
{ 
	"aliases":{
		"logingo": { }
	},
	"settings":{ 
		"number_of_shards":3, 
		"number_of_replicas":1 
	}, 
	"mappings":{ 
		"properties":{ 
			"PostDate":{ 
			"type":"date" 
			}, 
			"UserID":{ 
			"type":"long" 
			}, 
			"Username":{ 
			"type":"text" 
			} 
		} 
	} 
} 
`
	auditMappings = ` 
{ 
	"aliases":{
		"auditgo": { }
	},
	"settings":{ 
		"number_of_shards":3, 
		"number_of_replicas":1 
	}, 
	"mappings":{ 
		"properties":{ 
			"DateTime":{ 
			"type":"date" 
			}, 
			"Username":{ 
			"type":"text" 
			}, 
			"JsonModel":{ 
			"type":"text" 
			},
			"TableName":{ 
			"type":"text" 
			},
			"OperationName":{ 
			"type":"text" 
			} 
		} 
	} 
}
`
	errorMappings = ` 
{ 
	"aliases":{
		"errorgo": { }
	},
	"settings":{ 
		"number_of_shards":3, 
		"number_of_replicas":1 
	}, 
	"mappings":{ 
		"properties":{ 
			"DateTime":{ 
			"type":"date" 
			}, 
			"UserName":{ 
			"type":"text" 
			}, 
			"Message":{ 
			"type":"text" 
			},
			"ModuleName":{ 
			"type":"text" 
			}, 
			"ActionName":{ 
			"type":"text" 
			},
			"ErrorCode":{ 
			"type":"integer" 
			} 
		} 
	} 
}
`
)

var ElasticMaps = map[string]string{
	shared.Config.ELASTICLOGININDEX: loginMappings,
	shared.Config.ELASTICAUDITINDEX: auditMappings,
	shared.Config.ELASTICERRORINDEX: errorMappings,
}
