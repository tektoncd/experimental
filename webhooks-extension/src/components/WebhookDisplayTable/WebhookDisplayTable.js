import React, { Component } from 'react';
import { Link, Redirect } from 'react-router-dom'

import './WebhookDisplayTable.scss';
import Delete from '@carbon/icons-react/lib/delete/16';
import AddAlt16 from '@carbon/icons-react/lib/add--alt/16';
import { getWebhooks } from '../../api'

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

  constructor(props) {
    super(props);

    this.state = {
      error: null,
      isLoaded: false,
      webhooks: [],
      showNotification: false,
      notificationErrorMessage: '',
      notificationStatus: '',
      notificationStatusMsgShort: ''
    };

  }

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
          notificationErrorMessage: "Failure occured fetching webhooks, error returned from the REST endpoint was : " + text,
          notificationStatus: 'error',
          notificationStatusMsgShort: 'Error:',
          showNotification: true,
        });
      });
    }
  }

  handleSelectedRows = (rows) => {
    console.log(rows);
  }

  formatCellContent(id, value) {
    // Render the git repo as a clickable link
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
          
          <div className="table-container">

            {this.props.showNotificationOnTable && (
              <InlineNotification 
                kind='success' 
                subtitle='' 
                title='Webhook created successfully.'>
              </InlineNotification>
            )}
            
            <DataTable
              rows={initialRows}
              headers={headers}
              render={({ rows,
                        headers,
                        getHeaderProps,
                        getRowProps,
                        getSelectionProps,
                        getBatchActionProps,
                        selectedRows,
                        onInputChange
              }) => (
                        <TableContainer title="Webhooks">
                          <TableToolbar id="toolbar">
                            <TableBatchActions {...getBatchActionProps()}>
                              <TableBatchAction id="delete-btn" renderIcon={Delete} onClick={() => { this.handleSelectedRows(selectedRows) }}>Delete</TableBatchAction>
                            </TableBatchActions>
                            
                            <TableToolbarContent>
                              <div className="search-bar">
                                <TableToolbarSearch onChange={onInputChange} />
                                </div>
                                    <div className="btn-div">
                                      <Button as={Link} id="create-btn" to={this.props.match.url + "/create"}>
                                        Add
                                        <div className="create-icon">
                                          <AddAlt16/>
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
                                  <TableHeader {...getHeaderProps({ header })} isSortable="true" isSortHeader="true">{header.header}</TableHeader>
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
        );
      }
    } else {
      if (this.state.showNotification) {
        return (
          <div>
            {this.state.showNotification && (
              <InlineNotification
                kind={this.state.notificationStatus}
                subtitle={this.state.notificationErrorMessage}
                title={this.state.notificationStatusMsgShort}>
              </InlineNotification>
            )}
          </div>
        )
      } else {
        return (
          <div className="spinner-div">
            <Loading withOverlay={false} active="true" className="loading-spinner" />
          </div>
        )
      }
    }
  }
}