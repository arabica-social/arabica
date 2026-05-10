[private]
default: style templ-generate run

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json go run ./cmd/server -known-dids known-dids.txt

templ-watch:
    @templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/server -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

integration-test:
    @cd tests/integration && go test -v ./... -count=1 

verbose-integration-test:
    @cd tests/integration && INTEGRATION_LOGS=true go test -v ./... -count=1 

style: style-arabica

# style-oolong

style-arabica:
    @cat static/css/tokens.css static/css/reset.css static/css/utilities.css static/css/components.css > static/css/output.css

style-oolong:
    @cat static/css/tokens.css static/css/reset.css static/css/utilities.css static/css/components.css static/css/themes/oolong.css > static/css/output-oolong.css
