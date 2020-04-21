#!/usr/bin/env sh

# This script is meant to be run inside a container running alpine image

set -euo pipefail

apk add -u git tree

EXIT_CODE=0
LAST_RELEASE_HASH=$(git merge-base master HEAD)

cd helm

echo "📝 Checking for updated chart versions"

for chart in */; do
    ## If LAST_RELEASE_HASH does not include the chart, then it's a new chart and does not need a version increment

    TREE=$(git ls-tree -d $LAST_RELEASE_HASH $chart)

    if [[ -z "$TREE" ]]; then
        echo "✅ Chart $chart is a new chart since the last release"
        continue
    fi

    ## If no DIFF since LAST_RELEASE_HASH then it has not been modified 

    DIFF=$(git --no-pager diff master...HEAD -- $chart)

    if [[ -z "$DIFF" ]]; then
        echo "✅ Chart $chart had no changes since the last release"
        continue
    fi

    LAST_COMMIT_HASH=$(git --no-pager log --pretty=tformat:"%H" -- $chart | awk 'FNR <= 1')
    LAST_RELEASE_CHART_VERSION=$(git --no-pager show $LAST_RELEASE_HASH:helm/"$chart"Chart.yaml | grep 'version:' | xargs | cut -d' ' -f2 | tr -d '[:space:]')
    LAST_COMMIT_CHART_VERSION=$(git --no-pager show $LAST_COMMIT_HASH:helm/"$chart"Chart.yaml | grep 'version:' | xargs | cut -d' ' -f2 | tr -d '[:space:]')

    if [[ $LAST_RELEASE_CHART_VERSION == $LAST_COMMIT_CHART_VERSION ]]; then
        echo "❌ Chart $chart has the same Chart version as the last release $LAST_COMMIT_CHART_VERSION"
        EXIT_CODE=1
    else 
        echo "✅ Chart $chart has a different version since the last release ($LAST_RELEASE_CHART_VERSION -> $LAST_COMMIT_CHART_VERSION)"
    fi
done

exit $EXIT_CODE

