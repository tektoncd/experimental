import React, { Component } from 'react';
import { withRouter } from 'react-router-dom';

import { Button, TextInput, Dropdown, Form, Tooltip, DropdownSkeleton, Modal, InlineNotification, TooltipIcon } from 'carbon-components-react';
import { getSecrets, createWebhook, createSecret, deleteSecret } from '../../api/index';

import AddAlt20 from '@carbon/icons-react/lib/add--alt/20';
import SubtractAlt20 from '@carbon/icons-react/lib/subtract--alt/20';
import ViewFilled from '@carbon/icons-react/lib/view--filled/20';
import ViewOffFilled from '@carbon/icons-react/lib/view--off--filled/20';
import Infomation from "@carbon/icons-react/lib/information/16";

import './WebhookCreate.scss';

function validateInputs(value, id) {

  const trimmed = value.trim();

  if (trimmed === "") {
    return false;
  }

  if (id === "name" || id === "newSecretName") {
    if (trimmed.length > 253) {
      return false;
    }

    if (/[^-.a-z1-9]/.test(trimmed)) {
      return false;
    }
  }

  return true;
}

function invalidFieldsLocator(fields, name, value) {
  const newInvalidFields = fields;
  const idIndex = newInvalidFields.indexOf(name);
  if (validateInputs(value, name)) {
    if (idIndex !== -1) {
      newInvalidFields.splice(idIndex, 1);
    }
  } else if (idIndex === -1) {
    newInvalidFields.push(name);
  }

  return newInvalidFields;
}

const CustomTooltip = props => (
  <TooltipIcon {...props} >
    <Infomation />
  </TooltipIcon>
);

CustomTooltip.defaultProps = {
  onClick: e => e.preventDefault()
};

class WebhookCreatePage extends Component {

  constructor(props) {
    super(props);

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
      apiSecrets: '',
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
      // toggle access token 'password' or 'text'
      pwType: 'password',
      visibleCSS: 'token-visible',
      invisibleCSS: 'token-invisible',
      //array storing invalid inputs
      invalidFields: []
    };
  }

  componentDidMount(){
    // turn off webhook created message
    this.props.setShowNotificationOnTable(false);
    this.fetchSecrets();
    if(this.props.showLastWebhookDeletedNotification){
      this.setState({
        notificationMessage: "Last webhook(s) deleted successfully.",
        notificationStatus: 'success',
        notificationStatusMsgShort: 'Success:',
        showNotification: true
      });
      this.props.setshowLastWebhookDeletedNotification(false);
    }
    if(this.props.pipelinesErrorMessage){
      this.setState({
        notificationMessage: this.props.pipelinesErrorMessage,
        notificationStatus: 'error',
        notificationStatusMsgShort: 'Error:',
        showNotification: true
      });
    }
    if(this.props.serviceAccountsErrorMessage){
      this.setState({
        notificationMessage: this.props.serviceAccountsErrorMessage,
        notificationStatus: 'error',
        notificationStatusMsgShort: 'Error:',
        showNotification: true
      });
    }
    if (this.isDisabled()) {
      document.getElementById("pipeline").firstElementChild.tabIndex = -1;
      document.getElementById(
        "serviceAccounts"
      ).firstElementChild.tabIndex = -1;
    }
  }

  componentDidUpdate() {
    if (this.props.namespaces) {
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
  }

  async fetchSecrets() {
    let s;
    try {
      s = await getSecrets();
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

  handleChange = (event) => {
    const {target} = event;
    const value = target.value;
    const name = target.name;
    this.setState(prevState => {
      const newInvalidFields = invalidFieldsLocator(prevState.invalidFields, name, value);
      return { [name]: value, invalidFields: newInvalidFields };
    });
  }

  handleChangeNamespace = (itemText) => {
    this.setState({
      namespace: itemText.selectedItem,
      pipeline: '',
      serviceAccount: '',
    });
    if (!this.state.pipelineFail) {
      this.props.fetchPipelines(itemText.selectedItem);
    }
    if (!this.state.serviceAccountsFail) {
      this.props.fetchServiceAccounts(itemText.selectedItem);
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

    let invalidFields = [];

    const {
      gitsecret,
      dockerRegistry,
      repository,
      name,
      namespace,
      pipeline,
      serviceAccount
    } = this.state;

    if (!validateInputs(name, "name")) {
      invalidFields.push("name");
    }

    if (!validateInputs(dockerRegistry, "dockerRegistry")) {
      invalidFields.push("name");
    }

    if (!validateInputs(repository, "repository")) {
      invalidFields.push("repository");
    }

    if (invalidFields.length === 0) {
      const requestBody = {
        name,
        gitrepositoryurl: repository,
        accesstoken: gitsecret,
        pipeline,
        namespace,
        serviceaccount: serviceAccount,
        dockerregistry: dockerRegistry
      };

      createWebhook(requestBody).then(() => {
        this.props.setShowNotificationOnTable(true);
        this.returnToTable();
      }).catch(error => {
         error.response.text().then((text) => {
          this.setState({
            notificationMessage: 'Failed to create webhook, error returned was : ' + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true
          });
        });
      });
    } else {
      this.setState({ invalidFields });
    }
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

  displayNamespaceDropDown = () => {
    if (this.props.isFetchingNamespaces) {
      return <DropdownSkeleton/>
    }
    return <Dropdown
        id="namespace"
        label="select namespace"
        items={this.props.namespaces}
        onChange={this.handleChangeNamespace}
      />
  }

  displayPipelineDropDown = () => {
    if (!this.isDisabled()) {
      if (this.props.isFetchingPipelines) {
        return <DropdownSkeleton />;
      }
    }
    const pipelineItems = this.props.pipelines
      .filter(pipeline => pipeline.metadata.namespace === this.state.namespace)
      .map(pipeline => pipeline.metadata.name);
    return (
      <Dropdown
        data-testid="pipelinesDropdown"
        id="pipeline"
        label="select pipeline"
        items={pipelineItems}
        disabled={this.isDisabled()}
        onChange={this.handleChangePipeline}
      />
    );
  };

  displaySecretDropDown = (secretItems) => {
    if (!this.state.apiSecrets) {
      return <DropdownSkeleton />
    }

    return <Dropdown
      id="git"
      label="select secret"
      items={secretItems}
      onChange={this.handleChangeSecret}
      selectedItem={this.state.gitsecret}
    />
  }

  displayServiceAccountDropDown = () => {
    if (!this.isDisabled()) {
      if (this.props.isFetchingServiceAccounts) {
        return <DropdownSkeleton />
      }
    }
    const saItems = this.props.serviceAccounts
      .filter(sa => sa.metadata.namespace === this.state.namespace)
      .map(sa => sa.metadata.name);
    return <Dropdown
      id="serviceAccounts"
      data-testid="serviceAccounts"
      label="select service account"
      items={saItems}
      disabled={this.isDisabled()}
      onChange={this.handleChangeServiceAcct}
    />
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
    deleteSecret(this.state.gitsecret).then(() => {
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
      this.fetchSecrets();
    })
  }

  createAccessTokenSecret = () => {
    const requestBody = {
      name: this.state.newSecretName,
      accesstoken: this.state.newTokenValue
    };
    createSecret(requestBody).then(() => {
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
      this.fetchSecrets();
    })
  }

  handleModalText = (event) => {
    if (event) {
      const target = event.target;
      const value = target.value;
      const name = target.name;
      this.setState(prevState => {
        const newInvalidFields = invalidFieldsLocator(prevState.invalidFields, name, value);
        return { [name]: value, invalidFields: newInvalidFields };
      });
    }
  }

  togglePasswordVisibility = () => {
    this.setState({
      pwType: this.state.pwType === 'password' ? 'text' : 'password',
      visibleCSS: this.state.visibleCSS === 'token-visible' ? 'token-invisible' : 'token-visible',
      invisibleCSS: this.state.invisibleCSS === 'token-invisible' ? 'token-visible' : 'token-invisible',
    });
  };

  handleNotificationClose = () => {
    this.setState({
      showNotification: false
    });
  }

  render() {

    const secretItems = [];
    const { invalidFields } = this.state;

    if (this.state.apiSecrets) {
      this.state.apiSecrets.map(function (secretResource, index) {
        secretItems[index] = secretResource['name'];
      });
    }

    return (

      <div className="webhook-create" data-testid="webhook-create">
        <div className="notification">
          {this.state.showNotification && (
            <InlineNotification
              kind={this.state.notificationStatus}
              subtitle={this.state.notificationMessage}
              title={this.state.notificationStatusMsgShort}
              lowContrast
              onCloseButtonClick={this.handleNotificationClose}
            >
            </InlineNotification>
          )}
          {this.state.showNotification && window.scrollTo(0,0)}
        </div>

        <div className="create-container">
          <Form onSubmit={this.handleSubmit}>
            <div className="title">Create Webhook</div>

            <div className="row" id="sectionTitle">
              <u>Webhook Settings</u>
              <div className="sectionDescription">
                These settings are used for creating the webhook.
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="name-tooltip">
                <CustomTooltip tooltipText="The display name for your webhook in this user interface." />
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
                    hideLabel
                    labelText="Display Name"
                    data-testid="display-name-entry"
                    invalid={invalidFields.indexOf('name') > -1}
                    invalidText="Must be less than 563 characters, contain only lowercase alphanumeric character, . or - ."
                  />
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="git-tooltip">
                <CustomTooltip tooltipText="The URL of the git repository to create the webhook on."/>
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
                    hideLabel
                    labelText="Repository"
                    data-testid="git-url-entry"
                    invalid={invalidFields.indexOf('repository') > -1}
                    invalidText="Required."
                  />
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="namespace-tooltip">
                <CustomTooltip tooltipText="The namespace to operate in."/>
              </div>
              <div className="item-label">
                <div className="createLabel">Access Token</div>
              </div>
              <div className="del-sec-btn"><SubtractAlt20 id="delete-secret-button" onClick={() => { this.toggleDeleteDialog() }}/></div>
              <div className="git-access-drop-down-div">
                <div className="createDropDown">
                  {this.displaySecretDropDown(secretItems)}
                </div>
              </div>
              <div className="add-sec-btn"><AddAlt20 id="create-secret-button" onClick={() => { this.toggleCreateDialog() }}/></div>
            </div>

            <div className="row"/>
            <div className="row" id="sectionTitle">
              <u>Target Pipeline Settings</u>
              <div className="sectionDescription">
                These settings select and configure the pipeline to execute when the webhook triggers.
              </div>
            </div>
            
            <div className="row">
              <div className="help-icon" id="pipeline-tooltip">
                <CustomTooltip tooltipText="The pipeline from the selected namespace to run when the webhook is triggered."/>
              </div>
              <div className="item-label">
                <div className="createLabel">Namespace</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayNamespaceDropDown()}
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="secret-tooltip">
                <CustomTooltip tooltipText="The kubernetes secret holding access information for the git repository. The credential must have sufficient privileges to create webhooks in the repository."/>
              </div>
              <div className="item-label">
                <div className="createLabel">Pipeline</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayPipelineDropDown()}
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="serviceaccount-tooltip">
                <CustomTooltip tooltipText="The service account under which to run the pipeline run. The service account needs to be patched with secrets to access both git and docker."/>
              </div>
              <div className="item-label">
                <div className="createLabel">Service Account</div>
              </div>
              <div className="entry-field">
                <div className="createDropDown">
                  {this.displayServiceAccountDropDown()}
                </div>
              </div>
            </div>

            <div className="row">
              <div className="help-icon" id="docker-tooltip">
                <CustomTooltip tooltipText="The docker registry to push images to."/>
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
                    invalid={invalidFields.indexOf('dockerRegistry') > -1}
                    invalidText="Required."
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
                <Button data-testid="cancel-button" id="cancel" onClick={() => { this.returnToTable() }}>Cancel</Button>
                <Button className="modal-btn" data-testid="create-button" type="submit" tabIndex={this.isFormIncomplete() ? -1 : 0} id={this.createButtonIDForCSS()} disabled={this.isFormIncomplete()}>Create</Button>
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
                  <Tooltip direction="bottom" triggerText="">
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
                    />
                  </div>
                </div>
              </div>

              <div className="modal-row">
                <div className="modal-row-help-icon">
                  <Tooltip direction="bottom" triggerText="">
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
                      invalid={invalidFields.indexOf('newTokenValue') > -1}
                      invalidText="Required."
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
