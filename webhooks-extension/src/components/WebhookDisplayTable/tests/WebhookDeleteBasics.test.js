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

const webhooks = [
  {
    id: '0|namespace',
    name: 'first test webhook',
    gitrepositoryurl: 'repo1',
    pipeline: 'pipeline1',
    namespace: 'namespace1'
  },
  {
    id: '2|namespace',
    name: 'second test webhook',
    gitrepositoryurl: 'repo2',
    pipeline: 'pipeline2',
    namespace: 'namespace2'
  },
  {
    id: '3|namespace',
    name: 'third test webhook',
    gitrepositoryurl: 'repo3',
    pipeline: 'pipeline3',
    namespace: 'namespace3'
  }
];

it('should show a delete button on webhook load', async () => {
  jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
  const { getByText, queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} />);
  expect(queryByTestId('webhook-notification')).toBeNull();
  await waitForElement(() => getByText('first test webhook'));
  const foundDeleteButton = document.getElementById('delete-btn');
  await waitForElement(() => foundDeleteButton);
});

it('should hide the delete modal on cancel click', async () => {
  jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
  const { getByText, queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} />);
  expect(queryByTestId('webhook-notification')).toBeNull();
  await waitForElement(() => getByText('first test webhook'));
  await waitForElement(() => getByText('second test webhook'));
  await waitForElement(() => getByText('third test webhook'));

  const foundDeleteButton = document.getElementById('delete-btn');
  await waitForElement(() => foundDeleteButton);
  fireEvent.click(foundDeleteButton);

  const foundDeleteButtonOnModal = document.getElementById('webhook-delete-modal').getElementsByClassName('bx--btn bx--btn--danger').item(0);
  const foundCancelButtonOnModal = document.getElementById('webhook-delete-modal').getElementsByClassName('bx--btn bx--btn--secondary').item(0);
  
  await waitForElement(() => foundDeleteButtonOnModal);
  await waitForElement(() => foundCancelButtonOnModal);

  fireEvent.click(foundCancelButtonOnModal) // We click the cancel button on the modal
  await waitForElement(() => getByText('Delete PipelineRuns?')) == false; // Delete dialog's now gone
});