import React, { Component } from 'react';
import { Route, withRouter } from "react-router-dom";
import { connect } from 'react-redux';

import { WebhookCreate } from './components/WebhookCreate';
import { WebhookDisplayTable } from './components/WebhookDisplayTable';

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
      getPipelinesErrorMessage,
      isFetchingNamespaces,
      isFetchingPipelines,
      match,
      namespace,
      namespaces,
      pipelines
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
              isFetchingNamespaces={isFetchingNamespaces}
              isFetchingPipelines={isFetchingPipelines}
              getPipelinesErrorMessage={getPipelinesErrorMessage}
              fetchPipelines={fetchPipelines}
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
    pipelines: props.selectors.getPipelines(state),
    isFetchingNamespaces: props.selectors.isFetchingNamespaces(state),
    isFetchingPipelines: props.selectors.isFetchingPipelines(state),
    getPipelinesErrorMessage: props.selectors.getPipelinesErrorMessage(state),
  };
}

const mapDispatchToProps = (dispatch, props) => ({
  fetchPipelines: namespace =>
    dispatch(props.actions.fetchPipelines({ namespace }))
});

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(withRouter(WebhooksApp));
