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

import React from 'react';
import { waitForElement, cleanup, fireEvent } from 'react-testing-library';
import { WebhookDisplayTable } from '../WebhookDisplayTable'
import * as API from '../../../api';
import { renderWithRouter } from '../../../test/utils/test'
import 'react-testing-library/cleanup-after-each';

beforeEach(() => {
  jest.restoreAllMocks
  jest.resetModules()
});

afterEach(() => {
  jest.clearAllMocks()
  cleanup()
});

function fakeDeleteWebhooksSuccess() {
  return {
    data: {},
    status: 204,
    statusText: 'OK',
    headers: {}
  };
}

function fakeDeleteWebhooksFailure() {
  return {
    data: {},
    status: 400,
    statusText: 'Error',
    headers: {},
    error: 'Error',
  };
}

const webhooks = [
  {
    id: '0|namespace',
    name: 'first test webhook',
    gitrepositoryurl: 'repo1',
    pipeline: 'pipeline1',
    namespace: 'namespace1'
  },
  {
    id: '1|namespace',
    name: 'second test webhook',
    gitrepositoryurl: 'repo2',
    pipeline: 'pipeline1',
    namespace: 'namespace1'
  }
];

it('should display a success message on a good delete', async () => {
  let getWebhooksMock = jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
  let deleteWebhooksMock = jest.spyOn(API, "deleteWebhooks").mockImplementation(() => Promise.resolve(fakeDeleteWebhooksSuccess));

  const { getByText, queryByTestId, getAllByLabelText } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*" setshowLastWebhookDeletedNotification={()=>{}} showNotificationOnTable={false}/>);

  expect(queryByTestId('webhook-notification')).toBeNull();

  await waitForElement(() => getByText('first test webhook'));

  const checkbox = getAllByLabelText('Select row')[0];
  await waitForElement(() => checkbox);

  fireEvent.click(checkbox);

  const foundDeleteButton = document.getElementById('delete-btn');
  await waitForElement(() => foundDeleteButton);

  fireEvent.click(foundDeleteButton);

  const foundDeleteButtonOnModal = document.getElementById('webhook-delete-modal').getElementsByClassName('bx--btn bx--btn--danger').item(0);
  await waitForElement(() => foundDeleteButtonOnModal);

  expect(document.getElementsByClassName('bx--inline-loading__text').length).toBe(0);

  fireEvent.click(foundDeleteButtonOnModal);

  //check notification present
  expect(document.getElementsByClassName('bx--inline-loading__text').length).toBe(1);
  expect(document.getElementsByClassName('bx--inline-loading__text')[0].innerHTML).toBe("Webhook(s)&nbsp;under&nbsp;deletion, please do not navigate away from this page...");


  expect(getWebhooksMock).toHaveBeenCalled();
  expect(deleteWebhooksMock).toHaveBeenCalled();

  await waitForElement(() => getByText('Webhook(s) deleted successfully.'));
});

it('should display a fail message on a bad delete', async () => {
  jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
  jest.spyOn(API, "deleteWebhooks").mockImplementation(() => Promise.reject(fakeDeleteWebhooksFailure));

  let { getByText, queryByTestId, getAllByLabelText } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*" setshowLastWebhookDeletedNotification={()=>{}} showNotificationOnTable={false}/>);

  expect(queryByTestId('webhook-notification')).toBeNull();

  await waitForElement(() => getByText('first test webhook'));

  const checkbox = getAllByLabelText('Select row')[0];
  await waitForElement(() => checkbox);

  fireEvent.click(checkbox);

  const foundDeleteButton = document.getElementById('delete-btn');
  await waitForElement(() => foundDeleteButton);
  fireEvent.click(foundDeleteButton);

  const foundDeleteButtonOnModal = document.getElementById('webhook-delete-modal').getElementsByClassName('bx--btn bx--btn--danger').item(0);
  await waitForElement(() => foundDeleteButtonOnModal);
  fireEvent.click(foundDeleteButtonOnModal);

  await waitForElement(() => getByText('An error occurred deleting webhook(s).'));
});
