import React from 'react';
import {useParams} from 'react-router';
import {connect} from 'react-redux';
import BasicDetail from './BasicDetail';
import {fetchTaskName} from '../redux/Actions/TaskActionName';
import {fetchTaskSuccess} from '../redux/Actions/TaskAction';
import Loader from '../loader/loader';
import './basicdetail.css';
import {API_URL} from '../../constants';


const Detail: React.FC = (props: any) => {
  const [newversion, setNewversion] = React.useState();
  const {taskId} = useParams();
  React.useEffect(() => {
    props.fetchTaskSuccess();
    fetch(`${ API_URL }/resource/${ taskId }/versions`)
      .then((response) => response.json())
      .then((TaskName) => setNewversion(TaskName));
    // eslint-disable-next-line
  }, []);


  if (props.TaskData && newversion !== undefined) {
    let temp: any = [];
    for (let i = 0; i < props.TaskData.length; i++) {
      if (props.TaskData[i].id === Number(taskId)) {
        if (props.TaskName) {
          (props.TaskData[i]).data = newversion.data;
          temp = newversion.data[newversion.data.length - 1];
        }
        if (temp.length === 0) {
          return (
            <div></div>
          );
        } else {
          return (
            < BasicDetail task={props.TaskData[i]}
              version={temp}
            />
          );
        }
      }
    }
  }

  return (
    <Loader />
  );
};
const mapStateToProps = (state: any) => ({
  TaskName: state.TaskName.TaskName,
  TaskData: state.TaskData.TaskData,
});
export default connect(mapStateToProps,
  {fetchTaskName, fetchTaskSuccess})(Detail);

