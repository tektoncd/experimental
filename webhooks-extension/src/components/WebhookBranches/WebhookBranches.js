/*
Copyright 2019 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import React, { Component } from 'react';
import { Modal } from 'carbon-components-react';
import { getDashboardAPIRoot } from '../../api';

import './WebhookBranches.scss';

import {
  DataTable,
  DataTableSkeleton,
  InlineNotification
} from 'carbon-components-react';

const {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableCell,
  TableHeader
} = DataTable;

export class WebhookBranches extends Component {
  constructor(props) {
    super(props);
    this.state = {
      rows: [],
      loading: true,
      error: null
    };
  }

  componentDidMount() {
    let { url, namespace, pipeline } = this.props.webhook;
    let [server, org, repo] = url
      .toLowerCase()
      .replace(/https?:\/\//, "")
      .split("/");
    
    if (repo != undefined && repo.endsWith(".git")) {
      repo = repo.substring(0, repo.length - 4);
    }

    this.props
      .fetchPipelineRuns(
        namespace,
        [
          `webhooks.tekton.dev/gitOrg=${org}`,
          `webhooks.tekton.dev/gitServer=${server}`,
          `webhooks.tekton.dev/gitRepo=${repo}`,
          `tekton.dev/pipeline=${pipeline}`
        ]
      )
      .then(pipelineRuns => {
        let branches = [];
        const rows = pipelineRuns
          .sort(
            (a, b) =>
              new Date(
                b.status.conditions[
                  b.status.conditions.length - 1
                ].lastTransitionTime
              ) -
              new Date(
                a.status.conditions[
                  a.status.conditions.length - 1
                ].lastTransitionTime
              )
          )
          .reduce((result, pipelineRun) => {
            if (
              pipelineRun.metadata.labels["webhooks.tekton.dev/gitBranch"] != undefined && branches.indexOf(pipelineRun.metadata.labels["webhooks.tekton.dev/gitBranch"]) === -1
            ) {
              branches.push(pipelineRun.metadata.labels["webhooks.tekton.dev/gitBranch"]);
              const time = new Date(
                pipelineRun.status.conditions[
                  pipelineRun.status.conditions.length - 1
                ].lastTransitionTime
              );
              result.push({
                id: `${pipelineRun.metadata.labels["webhooks.tekton.dev/gitBranch"]}-branch`,
                branch: pipelineRun.metadata.labels["webhooks.tekton.dev/gitBranch"],
                time: `${time.toLocaleDateString()} - ${time.toLocaleTimeString()}`,
                status:
                  pipelineRun.status.conditions[
                    pipelineRun.status.conditions.length - 1
                  ].reason
              });
            }
            return result;
          }, []);

        this.setState({
          rows,
          loading: false
        });
      })
      .catch(error => {
        error.response.text().then(text => {
          this.setState({
            error: text,
            rows: [],
            loading: false
          });
        });
      });
  }

  formatCellContent(id, value, ns, pipe, repourl) {
    // Render the branch as a clickable link
    let url = new URL(repourl);
    let server = url.hostname;
    let org = url.pathname.split('/')[1].toLowerCase()
    let repo = url.pathname.split('/')[2].toLowerCase()
    if (repo != undefined && repo.toLowerCase().endsWith(".git")) {
      repo = repo.substring(0, repo.length - 4);
    }
    if (id.endsWith(":branch")) {
      const dashboardAPIRoot = getDashboardAPIRoot();
      let uri = `${dashboardAPIRoot}/#/namespaces/${ns}/pipelineruns?labelSelector=tekton.dev%2Fpipeline%3D${pipe}%2Cwebhooks.tekton.dev%2FgitServer%3D${server}%2Cwebhooks.tekton.dev%2FgitOrg%3D${org}%2Cwebhooks.tekton.dev%2FgitRepo%3D${repo}%2Cwebhooks.tekton.dev%2FgitBranch%3D${value}`
      return <a href={uri} rel="noopener noreferrer">{value}</a>
    } else {
      return value
    }
  }

  render() {
    const { close } = this.props;
    const { rows, loading, error } = this.state;

    const headers = [
      {
        key: 'branch',
        header: 'Branch'
      },
      {
        key: 'time',
        header: 'Last Build Time'
      },
      {
        key: 'status',
        header: 'Status'
      }
    ];

    return (
      <Modal
        open
        id="webhook-branches-modal"
        modalHeading="Latest PipelineRuns By Branch:"
        passiveModal
        onRequestClose={close}
      >
        {error && (
          <InlineNotification
            kind="error"
            subtitle={error}
            title="Error:"
            lowContrast
          />
        )}
        <div className="WebhookDetails">
          <p>
            <span>Webhook Name: </span>
            {this.props.webhook.name}
          </p>
          <p>
            <span>Repository: </span>
            <a
              target="_blank"
              rel="noopener noreferrer"
              href={this.props.webhook.url}
            >
              {this.props.webhook.url}
            </a>
          </p>
          <p>
            <span>Pipeline: </span>
            {this.props.webhook.pipeline}
          </p>
          <p>
            <span>Namespace: </span>
            {this.props.webhook.namespace}
          </p>
        </div>
        <DataTable
          useZebraStyles
          rows={rows}
          headers={headers}
          render={({ rows, headers, getHeaderProps, getRowProps }) => (
            <TableContainer>
              {loading ? (
                <DataTableSkeleton
                  rowCount={1}
                  columnCount={headers.length}
                  data-testid="loading-table"
                />
              ) : (
                <Table>
                  <TableHead>
                    <TableRow>
                      {headers.map(header => (
                        <TableHeader
                          key={header.id}
                          {...getHeaderProps({ header })}
                          isSortable
                          isSortHeader
                        >
                          {header.header}
                        </TableHeader>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {rows.map(row => (
                      <TableRow {...getRowProps({ row })} key={row.id}>
                        {row.cells.map((cell, index) => (
                          <TableCell
                            className="cellText"
                            key={cell.id}
                            data-status={
                              index === row.cells.length - 1 ? cell.value : null
                            }
                          >
                            {this.formatCellContent(cell.id, cell.value, this.props.webhook.namespace, this.props.webhook.pipeline, this.props.webhook.url)}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </TableContainer>
          )}
        />
        {rows.length === 0 && !loading && (
          <div className="noBranches">
            <p>Unable to identify any PipelineRuns initiated by this webhook.</p>
            <a href="https://github.com/tektoncd/experimental/webhooks-extension/docs/Labels.md">Click here for help.</a>
          </div>
        )}
      </Modal>
    );
  }
}
