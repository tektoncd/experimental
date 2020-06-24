import React from 'react';
import Description from '../description/Description';
import {connect} from 'react-redux';
import {
  Flex,
  FlexItem,
} from '@patternfly/react-core';
import {useParams} from 'react-router';
import {fetchTaskDescription} from '../redux/Actions/TaskActionDescription';
import {fetchTaskName} from '../redux/Actions/TaskActionName';
import store from '../redux/store';
const Detail: React.FC = (props: any) => {
  const {taskId} = useParams();
  React.useEffect(() => {
    // this dispatch is to reset previous description in redux store
    store.dispatch({type: 'FETCH_TASK_DESCRIPTION', payload: ''});
    props.fetchTaskName(taskId);
    // eslint-disable-next-line
  }, []);

  const newLine = '\n';
  let taskDescription: string = '';
  let catalogTaskDescription: string = '';
  let yamlData: string = '';

  if (props.TaskDescription) {
    taskDescription = (props.TaskName);
    catalogTaskDescription = props.TaskDescription;
    yamlData = '```' + newLine + props.TaskYaml + '```';

    return (
      <div style={{marginTop: '3em'}}>
        <Flex breakpointMods={[{modifier: 'row', breakpoint: 'lg'}, {
          modifier:
            'column', breakpoint: 'sm',
        }]}>
          <FlexItem>
            <Description
              Description={catalogTaskDescription}
              Yaml={yamlData}
              userTaskDescription={taskDescription} />
          </FlexItem>
        </Flex>
      </div>
    );
  } else {
    return (
      <div />
    );
  }
};

const mapStateToProps = (state: any) => {
  return {
    TaskDescription: state.TaskDescription.TaskDescription,
    TaskYaml: state.TaskYaml.TaskYaml,
    TaskName: state.TaskName.TaskName,
  };
};

export default connect(mapStateToProps,
  {fetchTaskDescription, fetchTaskName})(Detail);

