package index

const (
	Claims       = "claims"
	ClaimType    = "claim"
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
