/* eslint-disable max-len */
import React from 'react';
import {connect} from 'react-redux';
import '@patternfly/react-core/dist/styles/base.css';
import './index.css';
import {
  InputGroup,
  Flex,
  TextInput,
} from '@patternfly/react-core';
import {fetchTaskSuccess} from '../redux/Actions/TaskAction';
import {fetchTaskList} from '../redux/Actions/TaskDataListAction';
import store from '../redux/store';
import fuzzysort from 'fuzzysort';
export interface TaskPropData {
  id: number;
  name: string,
  description: string,
  rating: number,
  tags: [],
  lastUpdatedAt: string;
  latestVersion;
  catalog: [],
  type: string,
}

const SearchBar: React.FC = (props: any) => {
  // const [sort, setSort] = useState('Name');
  React.useEffect(() => {
    props.fetchTaskSuccess();
    props.fetchTaskList();
    // eslint-disable-next-line
  }, []);
  // Getting all data from store
  const [text, setText] = React.useState('');

  const onTextChanged = (e: any) => {
    const value = e;
    const trimmedText = value.trim();
    if (trimmedText.length !== 0) {
      const filtered = fuzzysort.go(trimmedText, props.TaskDataList, {
        keys: ['name', 'latestVersion.displayName'],
      });
      const suggestions = filtered.map((resource: any) => resource.obj);
      store.dispatch({
        type: 'FETCH_TASK_SUCCESS', payload: suggestions.sort((first: any, second: any) =>
          first.name > second.name ? 1 : -1),
      });
    } else {
      store.dispatch({
        type: 'FETCH_TASK_SUCCESS', payload: props.TaskDataList.sort((first: any, second: any) =>
          first.name > second.name ? 1 : -1),
      });
    }
    setText(value);
  };

  const textValue = text;
  store.dispatch({
    type: 'SEARCH_TEXT',
    payload: textValue,
  });

  return (

    <div className="search">
      <Flex breakpointMods={[{modifier: 'flex-1', breakpoint: 'lg'}]}>

        <Flex breakpointMods={[{modifier: 'column', breakpoint: 'lg'}]}>
        </Flex>
        <React.Fragment>


          <InputGroup style={{width: '100%'}}>
            <div style={{width: '100%', boxShadow: 'rgba'}}>
              <TextInput aria-label="text input example" value={textValue} type="search"
                onChange={onTextChanged} placeholder="Search for task or pipeline"
                style={{padding: '10px 5px', height: '2.7em'}} />

            </div>
          </InputGroup>


        </React.Fragment>
      </Flex>
    </div>
  );
};

const mapStateToProps = (state: any) => ({
  TaskData: state.TaskData.TaskData,
  TaskDataList: state.TaskDataList.TaskDataList,
  ResourceList: state.ResourceList.ResourceList,

});

export default connect(mapStateToProps, {fetchTaskSuccess, fetchTaskList})(SearchBar);

