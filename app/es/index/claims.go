package index

const (
	// Claims is the name used for the claims index of elastic search
	Claims = "claims"
	// ClaimType is the name used for the type of documents stored in the claims index
	ClaimType = "claim"
	// ClaimMapping is the mapping used by lighthouse and is initialized if the claims index does not exist on startup.
	ClaimMapping = `
{
  "settings": {
    "number_of_shards": 1
  },
  "mappings": {
    "claim": {
      "properties": {
        "value": {
          "type": "nested"
        },
        "suggest_name": {
          "type": "completion"
        },
        "suggest_desc": {
          "type": "completion"
        },
        "transaction_time": {
          "type": "date"
        }
      }
    }
  }
}`
)
