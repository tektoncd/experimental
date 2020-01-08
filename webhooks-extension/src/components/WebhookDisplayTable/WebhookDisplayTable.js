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
import './WebhookDisplayTable.scss';
import '../WebhookCreate/WebhookCreate.scss';
import Delete from '@carbon/icons-react/lib/delete/16';
import AddAlt16 from '@carbon/icons-react/lib/add--alt/16';
import { Modal, Checkbox } from 'carbon-components-react';
import { getWebhooks, deleteWebhooks } from '../../api';

import { Link, Redirect } from 'react-router-dom';
import { WebhookBranches } from '../WebhookBranches';

import {
  Button,
  DataTable,
  DataTableSkeleton,
  TableSelectAll,
  TableSelectRow,
  TableToolbar,
  TableToolbarContent,
  TableBatchActions,
  TableBatchAction,
  TableToolbarSearch,
  InlineNotification,
  InlineLoading,
} from 'carbon-components-react';

const {
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableCell,
  TableHeader,
} = DataTable;

const ALL_NAMESPACES = "*";
var displayTimeoutWarningNotification = false;

export class WebhookDisplayTable extends Component {


  state = {
    showTable: true,
    showDeleteDialog: false,
    checked: false,
    userSelectedRows: [],
    error: null,
    isLoaded: false,
    webhooks: [],
    notificationMessage: "",
    notificationStatus: 'success',
    notificationStatusMsgShort: 'Webhook created successfully.',
    selectedWebhook: null,
    deletingWebhook: false,
    showNotificationOnTable: false,
    overlayClassName: 'overlay-disabled'
  };

  formatCellContent(id, value) {
    // Render the git repo as a clickable link
    if (id.endsWith(":repository")) {
      return <a href={value} target="_blank" rel="noopener noreferrer">{value}</a>
    } else {
      return value
    }
  }

  componentDidMount() { 
    this.fetchWebhooksForTable()
  }

  async fetchWebhooksForTable() {
    try { 
      const webhookData = await getWebhooks();
      this.setState({
        isLoaded: true,
        webhooks: webhookData
      });
    } catch (error) {
        error.response.text().then((text) => {
          this.setState({
            notificationMessage: "Failure occurred fetching webhooks, error returned from the REST endpoint was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotificationOnTable: true,
          });
        });
      }
  }

  showDeleteDialogHandlerInvisible = () => {
    this.setState({
      showDeleteDialog: false,
      batchActions: null,
    });
  }

  showDeleteDialogHandlerVisible = (rowsInput, batchActions) => {
    if (rowsInput.length > 0) {
      this.setState({
        showDeleteDialog: true,
        checked: false,
        userSelectedRows: rowsInput,
        batchActions: batchActions,
      });
    } else {
      this.setState({
        checked: false,
        notificationMessage: "Error occurred deleting webhooks - no webhook was selected in the table.",
        notificationStatus: 'error',
        notificationStatusMsgShort: 'Error:',
        showNotificationOnTable: true,
      });
    }
  }

  handleDeleteWebhook = () => {
    let deleteRuns = false;

    if (this.state.checked) {
        deleteRuns = true;
    }

    let rowsToUse = this.state.userSelectedRows;

    let deletePromises = [];

    deletePromises = rowsToUse.map(function(rowIDObject) {
      let id = rowIDObject.id;
      let theName = id.substring(0, id.indexOf('|'));
      let namespace = id.substring(id.indexOf('|') + 1, id.lastIndexOf('|'));
      let repository = id.substring(id.lastIndexOf('|') + 1, id.length);
      let response = deleteWebhooks(theName, namespace, repository, deleteRuns);
      let deletionTimeoutInMs = 120000;
      let theTimeout = new Promise((resolve, reject) => {
        setTimeout(function () {
          reject("Warning - timed out waiting to delete webhooks and potentially PipelineRuns: manually check that the resources you expect have been deleted successfully.")
          displayTimeoutWarningNotification = true;
        }, deletionTimeoutInMs);
      })
      let deleteWithinTimePromise = Promise.race([response, theTimeout]);
      return deleteWithinTimePromise;
    })

    this.setState({
      deletingWebhook: true,
      showNotificationOnTable: false,
      overlayClassName: 'overlay-enabled',
      showDeleteDialog: false,
    })

    Promise.all(deletePromises)
      .then(() => {
        if(this.state.webhooks.length - rowsToUse.length === 0){
          this.props.setshowLastWebhookDeletedNotification(true);
        }
        else {
          this.setState({
            showNotificationOnTable: true,
            notificationStatus: 'success',
            notificationStatusMsgShort: 'Webhook(s) deleted successfully.',
            notificationMessage: '',
            overlayClassName: 'overlay-disabled',
            deletingWebhook: false,
          });
        }
      })
      .catch(() => {
        let notificationStatusToSet = '';
        let notificationMessageToSet = '';
        let notificationFullMessage = '';

        if (displayTimeoutWarningNotification) {
          notificationStatusToSet = 'warning'
          notificationMessageToSet = 'Warning - timed out waiting to delete webhooks and potentially PipelineRuns: manually check the resources were deleted.';
        } else {
          notificationStatusToSet = 'error'
          notificationMessageToSet =  'An error occurred deleting webhook(s).';
          notificationFullMessage = 'Check the webhook(s) existed and both the dashboard and extension pods are healthy.';
        }
        this.setState({
          showNotificationOnTable: true,
          notificationStatus: notificationStatusToSet,
          notificationStatusMsgShort: notificationMessageToSet,
          notificationMessage: notificationFullMessage,
          overlayClassName: 'overlay-disabled',
          deletingWebhook: false,
        });
      })
      .finally(() => {
        this.fetchWebhooksForTable();
      });
  }

  togglePipelineRunClicked = () => {
		this.setState({
      checked: !this.state.checked
		});
  }

  viewBranches = value => {
    this.setState({
      selectedWebhook: value
    })
  }

  closeBranches = () => {
    this.setState({
      selectedWebhook: null
    })
  }

  render() {

      if (this.state.webhooks.length === 0 && this.state.isLoaded) {
        return (
          // There are no webhooks, so redirect to the create panel
          <Redirect to={this.props.match.url + "/create"} />
        )
      } else {
        const { selectedNamespace, fetchPipelineRuns } = this.props;
        // There are webhooks so display table
        const headers = [
          {
            key: 'name',
            header: 'Name'
          },
          {
            key: 'repository',
            header: 'Git Repository'
          },
          {
            key: 'pipeline',
            header: 'Pipeline'
          },
          {
            key: 'serviceaccount',
            header: 'Service Account'
          }
        ];

        if (selectedNamespace === ALL_NAMESPACES) {
          headers.push({
            key: 'namespace',
            header: 'Namespace'
          });
        }

        let initialRows = [];
        // Populate the data for the rows array from the data from the webhooks get request made on page load
        this.state.webhooks.forEach(function({ gitrepositoryurl, name, namespace, pipeline, serviceaccount}) {
          if (selectedNamespace === ALL_NAMESPACES || namespace === selectedNamespace) {
            let webhook = {
              id: name + "|" + namespace + "|" + gitrepositoryurl,
              name,
              pipeline: pipeline,
              repository: gitrepositoryurl,
              serviceaccount: serviceaccount
            }

            if (selectedNamespace === ALL_NAMESPACES) {
              webhook.namespace = namespace;
            }

            initialRows.push(webhook);
          }
        });

        return (
          <div>
            <div className="table-container" data-testid="table-container">

              {(this.props.showNotificationOnTable || this.state.showNotificationOnTable) && (
                <InlineNotification
                  data-testid='webhook-notification'
                  kind={this.state.notificationStatus}
                  subtitle={this.state.notificationMessage}
                  title={this.state.notificationStatusMsgShort}
                  lowContrast>
                </InlineNotification>
              )}
              {this.state.deletingWebhook && (
                <InlineNotification
                  kind='info'
                  subtitle={<InlineLoading
                    status='active'
                    iconDescription='Webhook(s) under deletion indicator'
                    description='Webhook(s) under deletion, please do not navigate away from this page...'
                  >
                  </InlineLoading>}
                  title=''
                  lowContrast
                >
                </InlineNotification>
              )}
              
              <div className={this.state.overlayClassName} ></div>
              <DataTable
                rows={initialRows}
                headers={headers}
                render={({ 
                  rows,
                  headers,
                  getHeaderProps,
                  getRowProps,
                  getSelectionProps,
                  getBatchActionProps,
                  selectedRows,
                  onInputChange
                }) => (
                  <TableContainer>
                    <div className="header">
                    <div className="header-title">
                        <h4 className="bx--data-table-header__title">Webhooks</h4>
                    </div>
                      <TableToolbarContent>
                        <div className="search-bar">
                          <TableToolbarSearch disabled={!this.state.isLoaded} onChange={onInputChange} />
                        </div>
                        <div className="add-div">
                          <Button disabled={!this.state.isLoaded} kind="ghost" as={Link} id="add-btn" to={this.props.match.url + "/create"}>
                            <div className="add-icon-div">
                              <AddAlt16 className="add-icon"/>
                            </div>
                            Add Webhook
                          </Button>
                        </div>
                        </TableToolbarContent>
                    </div>
                    <TableToolbar>
                      <TableBatchActions {...getBatchActionProps()}>
                        <TableBatchAction id="delete-btn" renderIcon={Delete} onClick={() => {this.showDeleteDialogHandlerVisible(selectedRows, getBatchActionProps())}}>Delete</TableBatchAction>
                      </TableBatchActions>
                    </TableToolbar>
                    {
                      !this.state.isLoaded ? (
                        <DataTableSkeleton rowCount={1} columnCount={headers.length} data-testid="loading-table"/>
                      ) : (
                        <Table className="bx--data-table--zebra">
                          <TableHead>
                            <TableRow>
                              <TableSelectAll {...getSelectionProps()} />
                              {headers.map(header => (
                                <TableHeader key={header.id} {...getHeaderProps({ header })} isSortable={true} isSortHeader={true}>{header.header}</TableHeader>
                              ))}
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {rows.map(row => (
                              <TableRow {...getRowProps({ row })} key={row.id}>
                                <TableSelectRow {...getSelectionProps({ row })} />
                                {row.cells.map((cell, index) => (
                                  <TableCell
                                    onClick={
                                      index === 0
                                        ? () => {
                                            this.viewBranches({
                                              name: row.cells[0].value,
                                              url: row.cells[1].value,
                                              namespace: selectedNamespace === ALL_NAMESPACES ? row.cells[4].value : selectedNamespace,
                                              pipeline: row.cells[2].value,
                                              serviceaccount: row.cells[3].value
                                            });
                                          }
                                        : null
                                    }
                                    className={
                                      index === 0
                                        ? "clickableNameURL"
                                        : null
                                    }
                                    key={cell.id}
                                  >
                                    {this.formatCellContent(cell.id, cell.value)}
                                  </TableCell>
                                ))}
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      )
                    }
                  </TableContainer>
                )}
              />
            </div>
             
            {initialRows.length === 0 && selectedNamespace !== ALL_NAMESPACES && (
                <p className="noWebhooks">
                  {`No webhooks created under namespace '${selectedNamespace}', click 'Add Webhook' button to add a new one.`}
                </p>
              )
            }
            <div className="modal-delete">
              <Modal open={this.state.showDeleteDialog}
                id='webhook-delete-modal'
                modalLabel=''
                modalHeading="Please confirm you want to delete the following webhook(s):"
                primaryButtonText="Delete"
                secondaryButtonText="Cancel"
                danger={true}
                onRequestSubmit={this.handleDeleteWebhook}
                onSecondarySubmit={this.showDeleteDialogHandlerInvisible}
                onRequestClose={this.showDeleteDialogHandlerInvisible}
              >
                <ul>
                  {this.state.userSelectedRows.map(row => {
                    const { id } = row;
                    return <li key={id}>{id.substring(0, id.indexOf('|'))}</li>;
                  })}
                </ul>
                <fieldset>
                  <legend className="modal-legend"><b>Delete Associated PipelineRuns?</b></legend>
                  <div className="checkbox-div">
                    <Checkbox
                      id="pipelinerun-checkbox"
                      labelText="Check here to indicate that PipelineRuns associated with this webhook should also be deleted."
                      checked={this.state.checked}
                      onChange={this.togglePipelineRunClicked}
                      />
                    </div>
                </fieldset>
              </Modal>
            </div>
            {this.state.selectedWebhook && <WebhookBranches fetchPipelineRuns={fetchPipelineRuns} webhook={this.state.selectedWebhook} close={this.closeBranches}/>}
          </div>
        );
      }
    }
  }
