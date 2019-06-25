import React, { Component } from 'react';
import { withRouter } from 'react-router-dom';

import { Button, TextInput, Dropdown, Form, Tooltip, DropdownSkeleton, Modal, InlineNotification } from 'carbon-components-react';
import { getNamespaces, getPipelines, getSecrets, getServiceAccounts, createWebhook, createSecret, deleteSecret } from '../../api/index';

import AddAlt20 from '@carbon/icons-react/lib/add--alt/20';
import SubtractAlt20 from '@carbon/icons-react/lib/subtract--alt/20';
import ViewFilled from '@carbon/icons-react/lib/view--filled/20';
import ViewOffFilled from '@carbon/icons-react/lib/view--off--filled/20';

import './WebhookCreate.scss';

class WebhookCreatePage extends Component {

  constructor(props) {
    super(props);
    // turn off webhook created message
    this.props.setShowNotificationOnTable(false);

    this.state = {
      // variables to stop re-attempts to load
      namespaceFail: false,
      pipelineFail: false,
      secretsFail: false,
      serviceAccountsFail: false,
      // selected values stored as these
      name: '',
      repository: '',
      namespace: '',
      pipeline: '',
      gitsecret: '',
      serviceAccount: '',
      dockerRegistry: '',
      // fetched data from api calls
      apiNamespaces: '',
      apiPipelines: '',
      apiSecrets: '',
      apiServiceAccounts: '',
      // whether or not to show secret modals
      showDeleteDialog: false,
      showCreateDialog: false,
      // whether or not to show error for delete with no secret selected
      showNotification: false,
      // error messages
      notificationMessage: "",
      notificationStatus: 'success', // or error, or warning,
      notificationStatusMsgShort: 'Secret deleted successfully',
      // create secret vars
      newSecretName: '',
      newTokenValue: '',
      createSecretDisabled: true,
      // toggle access token 'password' or 'text'
      pwType: 'password',
      visibleCSS: 'token-visible',
      invisibleCSS: 'token-invisible'
    };
  }

  async fetchNamespaces() {
    let ns;
    try {
      ns = await getNamespaces();
      this.setState({apiNamespaces: ns})
    } catch (error) {
        error.response.text().then((text) => {
          this.setState({
            namespaceFail: true,
            notificationMessage: "Failed to get namespaces, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true,
          });
        });
    }
  }

  async fetchPipelines(namespace) {
    let pl;
    try {
      pl = await getPipelines(namespace);
      this.setState({apiPipelines: pl})
    } catch (error) {
        error.response.text().then((text) => {
          this.setState({
            pipelineFail: true,
            notificationMessage: "Failed to get pipelines, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true,
          });
        });
    }
  }
  
  async fetchSecrets(namespace) {
    let s;
    try {
      s = await getSecrets(namespace);
      this.setState({apiSecrets: s})
    } catch (error) {
        error.response.text().then((text) => {
          this.setState({
            secretsFail: true,
            notificationMessage: "Failed to get secrets, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true,
          });
        });
    }
  }

  async fetchServiceAccounts(namespace) {
    let sa;
    try {
      sa = await getServiceAccounts(namespace);
      this.setState({apiServiceAccounts: sa})
    } catch (error) {
        error.response.text().then((text) => {
          this.setState({
            serviceAccountsFail: true,
            notificationMessage: "Failed to get service accounts, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true,
          });
        });
    }
  }

  handleChange = (event) => {
    const target = event.target;
    const value = target.value;
    const name = target.name;
    this.setState({[name]: value});
  }

  handleChangeNamespace = (itemText) => {
    this.setState({
      namespace: itemText.selectedItem,
      apiPipelines: '',
      apiSecrets: '',
      apiServiceAccounts: '',
      pipeline: '',
      gitsecret: '',
      serviceAccount: '',
    });
    if (!this.state.pipelineFail) {
      this.fetchPipelines(itemText.selectedItem);
    }
    if (!this.state.secretsFail) {
      this.fetchSecrets(itemText.selectedItem);
    }
    if (!this.state.serviceAccountsFail) {
      this.fetchServiceAccounts(itemText.selectedItem);
    }
  }

  handleChangePipeline = (itemText) => {
    this.setState({pipeline: itemText.selectedItem });
  }

  handleChangeSecret = (itemText) => {
    this.setState({gitsecret: itemText.selectedItem });
  }

  handleChangeServiceAcct = (itemText) => {
    this.setState({serviceAccount: itemText.selectedItem });
  }

  handleSubmit = (e) => {
    e.preventDefault();
    
    const requestBody = {
      name: this.state.name,
      gitrepositoryurl: this.state.repository,
      accesstoken: this.state.gitsecret,
      pipeline: this.state.pipeline,
      namespace: this.state.namespace,
      serviceaccount: this.state.serviceAccount,
      dockerregistry: this.state.dockerRegistry
    };

    createWebhook(requestBody).then(() => {
      this.props.setShowNotificationOnTable(true);
      this.returnToTable();
    }).catch(error => {
       error.response.text().then((text) => {
        this.setState({
          notificationMessage: "Failed to create webhook, error returned was : " + text,
          notificationStatus: 'error',
          notificationStatusMsgShort: 'Error:',
          showNotification: true,
        });
      });
    });
  }


  returnToTable = () => {
    const cutpoint = this.props.match.url.lastIndexOf('/');
    const matchURL = this.props.match.url.slice(0, cutpoint);
    this.props.history.push(matchURL);
  }

  isDisabled = () => {
    if (this.state.namespace === "") {
      return true;
    }
    return false
  }

  isFormIncomplete = () => {
    if (!this.state.name || !this.state.repository || !this.state.namespace ||
      !this.state.pipeline || !this.state.gitsecret || !this.state.serviceAccount ||
      !this.state.dockerRegistry ) {
        return true;
    }
    return false
  }

  createButtonIDForCSS = () => {
    if (this.isFormIncomplete()) {
      return "disable"
    }
    return "submit"
  }

  displayNamespaceDropDown = (namespaceItems) => {
    if (!this.state.apiNamespaces) {
      return <DropdownSkeleton/>
    }
    return <Dropdown
        id="namespace"
        label="select namespace"
        items={namespaceItems}
        tabIndex={5}
        onChange={this.handleChangeNamespace}
      />
  }

  displayPipelineDropDown = (pipelineItems) => {
    if (!this.isDisabled()) {
      if (!this.state.apiPipelines) {
        return <DropdownSkeleton />
      }
    }
    return <Dropdown
      id="pipeline"
      label="select pipeline"
      items={pipelineItems}
      tabIndex={7}
      disabled={this.isDisabled()}
      onChange={this.handleChangePipeline}
    />
  }

  displaySecretDropDown = (secretItems) => {
    if (!this.isDisabled()) {
      if (!this.state.apiSecrets) {
        return <DropdownSkeleton />
      }
    }
    return <Dropdown
      id="git"
      label="select secret"
      items={secretItems}
      tabIndex={9}
      disabled={this.isDisabled()}
      onChange={this.handleChangeSecret}
      selectedItem={this.state.gitsecret}
    />
  }

  displayServiceAccountDropDown = (saItems) => {
    if (!this.isDisabled()) {
      if (!this.state.apiServiceAccounts) {
        return <DropdownSkeleton />
      }
    }
    return <Dropdown
      id="serviceAccounts"
      label="select service account"
      items={saItems}
      tabIndex={11}
      disabled={this.isDisabled()}
      onChange={this.handleChangeServiceAcct}
    />
  }

  getSecretButtonCSSID = () => {
    if (this.isDisabled()) {
      return "secButtonDisabled"
    }
    return "secButtonEnabled"
  }

  toggleDeleteDialog = () => {
    if (this.state.gitsecret) {
      let invert = !this.state.showDeleteDialog;
      this.setState({
        showDeleteDialog: invert,
        showNotification: false
      });
    } else {
      this.setState({
        showNotification: true,
        notificationMessage: "No secret selected. A secret must be selected from the drop down before selecting delete.",
        notificationStatus: "error",
        notificationStatusMsgShort: "Error:"
      })
    }
  }

  toggleCreateDialog = () => {
    if (this.state.showNotification) {
      this.setState({
        showNotification: false
      })
    }
    let invert = !this.state.showCreateDialog;
    this.setState({
      showCreateDialog: invert
    });
  }

  deleteAccessTokenSecret = () => {
    deleteSecret(this.state.gitsecret, this.state.namespace).then(() => {
      this.toggleDeleteDialog();
      this.setState({
        apiSecrets: '',
        gitsecret: '',
        showNotification: true,
        notificationMessage: "",
        notificationStatus: "success",
        notificationStatusMsgShort: "Secret deleted."
      });
    }).catch(error => {
      error.response.text().then((text) => {
        this.toggleDeleteDialog();
        this.setState({
          notificationMessage: "Failed to delete secret, error returned was : " + text,
          notificationStatus: 'error',
          notificationStatusMsgShort: 'Error:',
          showNotification: true,
        });
      });
    }).finally(() => {
      this.fetchSecrets(this.state.namespace);
    })
  }

  createAccessTokenSecret = () => {
    const requestBody = {
      name: this.state.newSecretName,
      accesstoken: this.state.newTokenValue
    };
    createSecret(requestBody, this.state.namespace).then(() => {
      this.toggleCreateDialog()
      this.setState({
        gitsecret: this.state.newSecretName,
        newSecretName: '',
        newTokenValue: '',
        showNotification: true,
        notificationMessage: "",
        notificationStatus: "success",
        notificationStatusMsgShort: "Secret created."
      });
    }).catch(error => {
      error.response.text().then((text) => {
        this.toggleCreateDialog()
        this.setState({
          newSecretName: '',
          newTokenValue: '',
          notificationMessage: "Failed to create secret, error returned was : " + text,
          notificationStatus: 'error',
          notificationStatusMsgShort: 'Error:',
          showNotification: true,
        });
      });
    }).finally(() => {
      this.fetchSecrets(this.state.namespace);
    })
  }

  handleModalText = (event) => {
    if (event) {
      const target = event.target;
      const value = target.value;
      const name = target.name;
      this.setState({ [name]: value });
    }
  }

  togglePasswordVisibility = () => {
    this.setState({
      pwType: this.state.pwType === 'password' ? 'text' : 'password',
      visibleCSS: this.state.visibleCSS === 'token-visible' ? 'token-invisible' : 'token-visible',
      invisibleCSS: this.state.invisibleCSS === 'token-invisible' ? 'token-visible' : 'token-invisible',
    });
  };

  render() {

    const namespaceItems = [];
    const pipelineItems = [];
    const secretItems = [];
    const saItems = [];

    if (!this.state.apiNamespaces) {
      if (!this.state.namespaceFail) {
        this.fetchNamespaces();
      }
    } else {
      this.state.apiNamespaces.items.map(function (namespaceResource, index) {
        namespaceItems[index] = namespaceResource.metadata['name'];
      });
      if (this.state.apiPipelines) {
        this.state.apiPipelines.items.map(function (pipelineResource, index) {
          pipelineItems[index] = pipelineResource.metadata['name'];
        });
      }
      if (this.state.apiSecrets) {
        this.state.apiSecrets.map(function (secretResource, index) {
          secretItems[index] = secretResource['name'];
        });
      }
      if (this.state.apiServiceAccounts) {
        this.state.apiServiceAccounts.items.map(function (saResource, index) {
          saItems[index] = saResource.metadata['name'];
        });
      }
      if (this.state.createSecretDisabled) {
        if (this.state.newSecretName && this.state.newTokenValue) {
          this.setState({
            createSecretDisabled: false
          })
        }
      } else {
        if (!this.state.newSecretName || !this.state.newTokenValue) {
          this.setState({
            createSecretDisabled: true
          })
        }
      }
    }

    return (
      
      <div className="webhook-create">
        <div className="notification">
          {this.state.showNotification && (
            <InlineNotification
              kind={this.state.notificationStatus}
              subtitle={this.state.notificationMessage}
              title={this.state.notificationStatusMsgShort}>
            </InlineNotification>
          )}
          {this.state.showNotification && window.scrollTo(0,0)}
        </div>
      
        <div className="create-container">
          <Form onSubmit={this.handleSubmit}>
            <div className="title">Create Webhook</div>

            <div className="row">
              <div className="help-icon" id="name-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={0}>
                  <p>The display name for your webhook in this user interface.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Name</div>
              </div>
              <div className="entry-field">
                <div className="createTextEntry">
                  <TextInput
                    id="id"
                    placeholder="Enter display name here"
                    name="name"
                    value={this.state.name}
                    onChange={this.handleChange}
                    tabIndex={1}
                    hideLabel
                    labelText="Display Name"
                    data-testid="display-name-entry"
                  />
                </div>
              </div>
            </div>

            <div className="row">
            <div className="help-icon" id="git-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={2}>
                  <p>The URL of the git repository to create the webhook on.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Repository URL</div>
              </div>
              <div className="entry-field">
                <div className="createTextEntry">
                  <TextInput
                    id="git-repo"
                    placeholder="https://github.com/org/repo.git"
                    name="repository"
                    value={this.state.repo}
                    onChange={this.handleChange}
                    tabIndex={3}
                    hideLabel
                    labelText="Repository"
                    data-testid="git-url-entry"
                  />
                </div>
              </div>
            </div>

            <div className="row">
            <div className="help-icon" id="namespace-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={4}>
                  <p>The namespace to operate in.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Namespace</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayNamespaceDropDown(namespaceItems)}
                </div>
              </div>
            </div>

            <div className="row">
            <div className="help-icon" id="pipeline-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={6}>
                  <p>The pipeline from the selected namespace to run when the webhook is triggered.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Pipeline</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayPipelineDropDown(pipelineItems)}
                </div>
              </div>
            </div>

            <div className="row">
            <div className="help-icon" id="secret-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={8}>
                  <p>The kubernetes secret holding access information for the git repository. The credential must have sufficient privileges to create webhooks in the repository.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Access Token</div>
              </div>
              <div className="del-sec-btn"><SubtractAlt20 id="delete-secret-button" className={this.getSecretButtonCSSID()} onClick={() => { this.toggleDeleteDialog() }}/></div>
              <div className="git-access-drop-down-div">
                <div className="createDropDown">
                  {this.displaySecretDropDown(secretItems)}
                </div>
              </div>
              <div className="add-sec-btn"><AddAlt20 id="create-secret-button" className={this.getSecretButtonCSSID()} onClick={() => { this.toggleCreateDialog() }}/></div>
            </div>

            <div className="row">
            <div className="help-icon" id="serviceaccount-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={10}>
                  <p>The service account under which to run the pipeline run.</p><br></br><p>The service account needs to be patched with secrets to access both git and docker.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Service Account</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayServiceAccountDropDown(saItems)}
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="docker-tooltip">
                <Tooltip direction="bottom" triggerText="" tabIndex={12}>
                  <p>The docker registry to push images to.</p>
                </Tooltip>
              </div>
              <div className="item-label">
                <div className="createLabel">Docker Registry</div>
              </div>
              <div className="entry-field">
                <div className="createTextEntry">
                  <TextInput
                    id="registry"
                    placeholder="Enter docker registry here"
                    name="dockerRegistry"
                    value={this.state.dockerRegistry}
                    onChange={this.handleChange}
                    hideLabel
                    labelText="Docker Registry"
                    data-testid="docker-reg-entry"
                  />
                </div>
              </div>
            </div>

            <div className="row">
            <div className="help-icon"></div>
              <div className="item-label"></div>
              <div className="entry-field"></div>
            </div>

            <div className="row">
            <div className="help-icon"></div>
              <div className="item-label"></div>
              <div className="entry-field">
                <Button data-testid="cancel-button" id="cancel" tabIndex={13} onClick={() => { this.returnToTable() }}>Cancel</Button>
                <Button data-testid="create-button" type="submit" tabIndex={14} id={this.createButtonIDForCSS()} disabled={this.isFormIncomplete()}>Create</Button>
              </div>
            </div>

          </Form>


          <div className="delete-modal">
            <Modal open={this.state.showDeleteDialog}
              id="delete-modal"
              modalLabel=""
              modalHeading="Please confirm you want to delete the following secret:" 
              primaryButtonText="Confirm"
              secondaryButtonText="Cancel"
              danger={false}
              onSecondarySubmit={() => this.toggleDeleteDialog()}
              onRequestSubmit={() => this.deleteAccessTokenSecret()}
              onRequestClose={() => this.toggleDeleteDialog()}>
              <div className="secret-to-delete">{this.state.gitsecret}</div>
            </Modal>
          </div>

          <div className="create-modal">
            <Modal open={this.state.showCreateDialog}
              id="create-modal"
              modalLabel=""
              modalHeading="" 
              primaryButtonText="Create"
              primaryButtonDisabled={this.state.createSecretDisabled}
              secondaryButtonText="Cancel"
              danger={false}
              onSecondarySubmit={() => this.toggleCreateDialog()}
              onRequestSubmit={() => this.createAccessTokenSecret()}
              onRequestClose={() => this.toggleCreateDialog()}>
            
              <div className="title">Create Access Token Secret</div>

              <div className="modal-row">
                <div className="modal-row-help-icon">
                  <Tooltip direction="bottom" triggerText="" tabIndex={15}>
                    <p>The name of the secret to create.</p>
                  </Tooltip>
                </div>
                <div className="modal-row-item-label">
                  <div>Name</div>
                </div>
                <div className="modal-row-entry-field">
                  <div className="">
                    <TextInput
                      id="secretName"
                      placeholder="Enter secret name here"
                      name="newSecretName"
                      type="text"
                      value={this.state.newSecretName}
                      onChange={this.handleModalText}
                      hideLabel
                      labelText="Secret Name"
                      tabIndex={16}
                    />
                  </div>
                </div>
              </div>

              <div className="modal-row">
                <div className="modal-row-help-icon">
                  <Tooltip direction="bottom" triggerText="" tabIndex={17}>
                    <p>The access token.</p>
                  </Tooltip>
                </div>
                <div className="modal-row-item-label">
                  <div>Access Token</div>
                </div>
                <div className="modal-row-entry-field">
                  <div className="token">
                    <TextInput
                      id="tokenValue"
                      placeholder="Enter access token here"
                      name="newTokenValue"
                      type={this.state.pwType}
                      value={this.state.newTokenValue}
                      onChange={this.handleModalText}
                      hideLabel
                      labelText="Access Token"
                      tabIndex={18}
                    />
                    <ViewFilled id="token-visible-svg" className={this.state.visibleCSS} onClick={this.togglePasswordVisibility} />
                    <ViewOffFilled id="token-invisible-svg" className={this.state.invisibleCSS} onClick={this.togglePasswordVisibility} />
                  </div>
                </div>
              </div>
            </Modal>
          </div>
        </div>
      </div>
    );
  }
}

export const WebhookCreate = withRouter(WebhookCreatePage)