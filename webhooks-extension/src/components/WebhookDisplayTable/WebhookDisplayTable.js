import React, { Component } from 'react';
import './WebhookDisplayTable.scss';
import Delete from '@carbon/icons-react/lib/delete/16';
import AddAlt16 from '@carbon/icons-react/lib/add--alt/16';
import { Modal } from 'carbon-components-react';
import { Checkbox } from './StyledCheckbox.js'
import { getWebhooks, deleteWebhooks, getSelectedRows } from '../../api';

import { Link, Redirect } from 'react-router-dom'; 

import {
  Button,
  DataTable,
  TableSelectAll,
  TableSelectRow,
  TableToolbar,
  TableToolbarContent,
  TableBatchActions,
  TableBatchAction,
  TableToolbarSearch,
  Loading,
  InlineNotification,
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
  };

  async componentDidMount() {
    let data;
    try {
      data = await getWebhooks();
      this.setState({
        isLoaded: true,
        webhooks: data
      });
    } catch (error) {
      error.response.text().then((text) => {
        this.setState({
          notificationMessage: "Failure occured fetching webhooks, error returned from the REST endpoint was : " + text,
          notificationStatus: 'error',
          notificationStatusMsgShort: 'Error:',
          showNotificationOnTable: true,
        });
      });
    }
  }

  formatCellContent(id, value) {
    // Render the git repo as a clickable link
    if (id.endsWith(":repository")) {
      return <a href={value} target="_blank">{value}</a>
    } else {
      return value
    }
  }

  componentDidMount() {
    this.getWebhooks();
  }

  async getWebhooks() {
    const webhookData = await getWebhooks();
    this.setState({
      isLoaded: true,
      webhooks: webhookData
    });
  }

  showDeleteDialogHandlerInvisible = () => {
    this.setState({
      showDeleteDialog: false
    });
  }

  showDeleteDialogHandlerVisible = rowsInput => {
		this.setState({
			showDeleteDialog: true,
      userSelectedRows: rowsInput,
    });
  }

  handleDeleteWebhook = () => {
    let deleteRuns = false;

    if (this.state.checked) {
        deleteRuns = true;
    };

    let rowsToUse = getSelectedRows(this.state.userSelectedRows);

    let deletePromises = [];

    deletePromises = rowsToUse.map(function(rowIDObject, indexInTheMap) {
      let id = rowIDObject.id;
      let theName = id.substring(0, id.lastIndexOf('|'));
      let namespace = id.substring(id.lastIndexOf('|') + 1, id.length);
      let response = deleteWebhooks(theName, namespace, deleteRuns);
      // Potentially needs to change or be configurable based on how many webhooks there are
      let deletionTimeoutInMs = 500;
      let theTimeout = new Promise((resolve, reject) => {
        setTimeout(function () {
          //reject("Timed out: check the webhook(s) existed and both the dashboard and extension pods are healthy.")
          reject("Check the webhook(s) existed and both the dashboard and extension pods are healthy.")
        }, deletionTimeoutInMs);
      })
      let deleteWithinTimePromise = Promise.race([response, theTimeout]);
      return deleteWithinTimePromise;
    })

    Promise.all(deletePromises).then(values => {
      this.getWebhooks();
      this.setState({
        showNotificationOnTable: true,
        showDeleteDialog: false,
        notificationStatus: 'success',
        notificationStatusMsgShort: 'Webhook(s) deleted successfully.',
        notificationMessage: '',
      });
     }).catch(error => {
      console.log(`error is: ${error}`)
      this.setState({
        showNotificationOnTable: true,
        showDeleteDialog: false,
        notificationStatus: 'error',
        notificationStatusMsgShort: 'An error occurred deleting webhook(s).',
        notificationMessage: 'Check the webhook(s) existed and both the dashboard and extension pods are healthy.',
        /* Todo use the actual error message here and include correct mocking in tests, 
        This is the only realistic case for now and just seeing a "502 bad gateway" for example isn't useful */
      });
    });
  }

  togglePipelineRunClicked = event => {
		this.setState({
      checked: event.target.checked
		});	
  }

  getContent(id, value) {
    if (id.endsWith(":repository")) {
      return <a href={value} target="_blank">{value}</a>
    } else {
      return value
    }
  }

  render() {

    if (this.state.isLoaded) {
      if (!this.state.webhooks.length) {
        return (
          // There are no webhooks, so redirect to the create panel
          <Redirect to={this.props.match.url + "/create"} />
        )
      } else {
        // There are webhooks so display table
        const headers = [
          {
            key: 'name',
            header: 'Name',
          },
          {
            key: 'repository',
            header: 'Git Repository',
          },
          {
            key: 'pipeline',
            header: 'Pipeline',
          },
          {
            key: 'namespace',
            header: 'Namespace',
          }
        ];
    
        let initialRows = []
        // Populate the data for the rows array from the data from the webhooks get request made on page load
        this.state.webhooks.map(function (webhook, keyIndex) {
          initialRows[keyIndex] = {
            id: webhook['name']+"|"+webhook['namespace'],
            name: webhook['name'],
            repository: webhook['gitrepositoryurl'],
            pipeline: webhook['pipeline'],
            namespace: webhook['namespace'],
          }
        })
        
        return (
          <div>
            <div className="table-container">

              {(this.props.showNotificationOnTable || this.state.showNotificationOnTable) && (              
                <InlineNotification
                  data-testid='webhook-notification'
                  kind={this.state.notificationStatus}
                  subtitle={this.state.notificationMessage}
                  title={this.state.notificationStatusMsgShort}>
                </InlineNotification>
              )}
              
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
                  <TableContainer title="Webhooks">
                    <TableToolbar>
                      <TableBatchActions {...getBatchActionProps()}>
                        <TableBatchAction id="delete-btn" renderIcon={Delete} onClick={() => {this.showDeleteDialogHandlerVisible(selectedRows)}}>Delete</TableBatchAction>
                      </TableBatchActions>

                      <TableToolbarContent>
                        <div className="search-bar">
                          <TableToolbarSearch onChange={onInputChange} />
                          </div>
                              <div className="btn-div">
                                <Button as={Link} id="create-btn" to={this.props.match.url + "/create"}>
                                  Add
                                  <div className="create-icon">
                                    <AddAlt16 />
                                  </div>
                                </Button>
                          </div>
                      </TableToolbarContent>

                    </TableToolbar>
          
                    <Table className="bx--data-table--zebra">
                      <TableHead>
                        <TableRow>
                          <TableSelectAll {...getSelectionProps()} />
                          {headers.map(header => (
                            <TableHeader {...getHeaderProps({ header })} isSortable={true} isSortHeader={true}>{header.header}</TableHeader>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {rows.map(row => (
                          <TableRow {...getRowProps({ row })} key={row.id}>
                            <TableSelectRow {...getSelectionProps({ row })} />
                            {row.cells.map(cell => (
                              <TableCell key={cell.id}>{this.formatCellContent(cell.id, cell.value)}</TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              />
            </div>

            <div className="modal">
              <Modal open={this.state.showDeleteDialog}
                id='webhook-delete-modal'
                modalLabel=''
                modalHeading="Confirm deletion of webhook(s)?"
                primaryButtonText="Delete"
                secondaryButtonText="Cancel"
                danger={true}
                onRequestSubmit={this.handleDeleteWebhook}
                onSecondarySubmit={this.showDeleteDialogHandlerInvisible}
                onRequestClose={this.showDeleteDialogHandlerInvisible}
                >
              <label>
                <p>Selecting the checkbox below will cause the deletion of associated PipelineRuns</p>
                <Checkbox
                  checked={this.state.checked}
                  onChange={this.togglePipelineRunClicked}
                />
                <span>Delete PipelineRuns?</span>
              </label>
              </Modal>
            </div>
          </div>
        );
      }
    } else {
      return(
        <div className="spinner-div"><Loading withOverlay={false} active className="loading-spinner" /></div>
      )
    }
  }
}
