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
  const [allversion, setAllversion] = React.useState();
  const {taskId} = useParams();
  React.useEffect(() => {
    props.fetchTaskSuccess();
    fetch(`${ API_URL }/resource/${ taskId }/versions`)
      .then((response) => response.json())
      .then((res) => setAllversion(res.data.versions));
    // eslint-disable-next-line
  }, []);


  if (props.TaskData) {
    // let temp string;
    for (let i = 0; i < props.TaskData.length; i++) {
      if (props.TaskData[i].id === Number(taskId)) {
        return (
          < BasicDetail task={props.TaskData[i]}
            version={allversion}
          />
        );
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
