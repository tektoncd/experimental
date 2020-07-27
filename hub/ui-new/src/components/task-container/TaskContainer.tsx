import React from 'react';
import {connect} from 'react-redux';
import {
  Gallery,
  EmptyState,
  EmptyStateIcon,
  EmptyStateBody,
  EmptyStateVariant,
}
  from '@patternfly/react-core';
import Task from '../task/Task';
import {fetchTaskSuccess} from '../redux/Actions/TaskAction';
import './index.css';
import {CubesIcon} from '@patternfly/react-icons';
import Loader from '../loader/loader';
export interface TaskPropData {
  name: string,
  description: string,
  rating: number,
  downloads: number,
  yaml: string,
  tags: [],
}

const TaskContainer: React.FC = (props: any) => {
  let tempArr: any;
  React.useEffect(() => {
    fetchTaskSuccess();
    // eslint-disable-next-line
  }, []);
  if (props.TaskData === undefined) {
    return (
      <div className="loader">
        <Loader />
      </div>
    );
  }
  if (props.TaskData != null) {
    tempArr = props.TaskData;
  }

  if (tempArr.length === 0) {
    return (

      <div style={{
        margin: 'auto',
      }}>
        <EmptyState variant={EmptyStateVariant.full}>
          <EmptyStateIcon icon={CubesIcon} />
          <EmptyStateBody>
            No match found.
          </EmptyStateBody>
        </EmptyState>
      </div>
    );
  }

  return (
    <div>

      <Gallery gutter="lg">

        {
          tempArr.map((task: any) => <Task key={task.id} task={task} />)
        }


      </Gallery>
    </div>
  );
};
const mapStateToProps = (state: any) => ({
  TaskData: state.TaskData.TaskData,
});
export default connect(mapStateToProps,
  {fetchTaskSuccess})(TaskContainer);
