
.PHONY: watch
watch:
	docker-compose up -d
	curl --request PUT --url http://localhost:9200/organizations | json_pp
	air