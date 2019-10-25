import React, { Component } from 'react';
import { Route, withRouter } from "react-router-dom";
import { connect } from 'react-redux';

import { WebhookCreate } from './components/WebhookCreate';
import { WebhookDisplayTable } from './components/WebhookDisplayTable';

const ALL_NAMESPACES = '*';

class WebhooksApp extends Component {

  constructor(props) {
    super(props);
    this.state = {
      showNotificationOnTable: false,
      showLastWebhookDeletedNotification: false
    }
    this.setShowNotificationOnTable = this.setShowNotificationOnTable.bind(this);
    this.setshowLastWebhookDeletedNotification = this.setshowLastWebhookDeletedNotification.bind(this);
  }

  setShowNotificationOnTable(value) {
    this.setState({showNotificationOnTable: value})
  }

  setshowLastWebhookDeletedNotification(value) {
    this.setState({showLastWebhookDeletedNotification: value})
  }

  render() {
    const {
      fetchPipelines,
      fetchPipelineRuns,
      fetchServiceAccounts,
      pipelinesErrorMessage,
      serviceAccountsErrorMessage,
      isFetchingNamespaces,
      isFetchingPipelines,
      isFetchingServiceAccounts,
      match,
      namespace,
      namespaces,
      pipelines,
      serviceAccounts
    } = this.props;

    return (
      <div>
        <Route
          exact
          path={`${match.path}/`}
          render={props => (
            <WebhookDisplayTable
              {...props}
              selectedNamespace={namespace}
              showNotificationOnTable={this.state.showNotificationOnTable}
              setshowLastWebhookDeletedNotification={
                this.setshowLastWebhookDeletedNotification
              }
              fetchPipelineRuns={fetchPipelineRuns}
            />
          )}
        />
        <Route
          path={`${match.path}/create`}
          render={props => (
            <WebhookCreate
              {...props}
              namespaces={namespaces}
              pipelines={pipelines}
              serviceAccounts={serviceAccounts}
              isFetchingNamespaces={isFetchingNamespaces}
              isFetchingPipelines={isFetchingPipelines}
              isFetchingServiceAccounts={isFetchingServiceAccounts}
              pipelinesErrorMessage={pipelinesErrorMessage}
              fetchPipelines={fetchPipelines}
              serviceAccountsErrorMessage={serviceAccountsErrorMessage}
              fetchServiceAccounts={fetchServiceAccounts}
              setShowNotificationOnTable={this.setShowNotificationOnTable}
              setshowLastWebhookDeletedNotification={
                this.setshowLastWebhookDeletedNotification
              }
              showLastWebhookDeletedNotification={
                this.state.showLastWebhookDeletedNotification
              }
            />
          )}
        />
      </div>
    );
  }
}

function mapStateToProps(state, props) {
  return {
    namespace: props.selectors.getSelectedNamespace(state),
    namespaces: props.selectors.getNamespaces(state),
    pipelines: props.selectors.getPipelines(state, { namespace: ALL_NAMESPACES }),
    isFetchingNamespaces: props.selectors.isFetchingNamespaces(state),
    isFetchingPipelines: props.selectors.isFetchingPipelines(state),
    pipelinesErrorMessage: props.selectors.getPipelinesErrorMessage(state),
    serviceAccountsErrorMessage: props.selectors.getServiceAccountsErrorMessage(state),
    isFetchingServiceAccounts: props.selectors.isFetchingServiceAccounts(state),
    serviceAccounts: props.selectors.getServiceAccounts(state, { namespace: ALL_NAMESPACES })
  };
}

const mapDispatchToProps = (dispatch, props) => ({
  fetchPipelines: namespace =>
    dispatch(props.actions.fetchPipelines({ namespace })),
  fetchServiceAccounts: namespace =>
    dispatch(props.actions.fetchServiceAccounts({ namespace })),
  fetchPipelineRuns: (namespace, filters) =>
    dispatch(props.actions.fetchPipelineRuns({namespace, filters}))
});

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(withRouter(WebhooksApp));
