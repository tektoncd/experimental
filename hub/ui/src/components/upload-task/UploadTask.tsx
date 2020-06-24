import React, {useState} from 'react';
import './UploadTask.css';
import {
  Link,
} from 'react-router-dom';
import {
  Form,
  FormGroup,
  TextInput,
  TextArea,
  ActionGroup,
  Button,
  ChipGroup,
  Chip,
  Dropdown,
  DropdownToggle,
  DropdownItem,
  Alert,
} from '@patternfly/react-core';
import {API_URL} from '../../constants';
const UploadTask: React.FC = () => {
  const intags: string[] = [];
  const [type, setType]=useState('Task');
  const [uploadMessage, setUploadMessage] = useState(' ');
  const [tags, setTags] = useState(intags);
  const [load, setLoad]=useState();
  const spiner:any = <div className="loading">
    Loading&#8230;
  </div>;
  // alert message for task upload
  let sendStatus:any='';
  const alertMessage=(status :any) =>{
    if (status['status'] === false) {
      sendStatus = <Alert variant="danger"
        isInline title={status['message']} />;
      setLoad('');
    } else {
      sendStatus = <Alert variant="success"
        isInline title={status['message']} />;
      setLoad('');
      setTimeout(() => window.location.assign('/'), 2000);
    }
    return sendStatus;
  };
  // / function for uloading task file
  const submitdata = (event: any) => {
    event.preventDefault();
    setLoad(spiner);
    const data = new FormData(event.target);
    const formdata = {
      name: data.get('task-name'),
      description: data.get('description'),
      type: type.toLowerCase(),
      tags: tags,
      github: data.get('tasklink'),
      user_id: Number(localStorage.getItem('usetrID')),
    };

    fetch(`${API_URL}/upload`, {
      method: 'POST',
      body: JSON.stringify(formdata),
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
    }).then((resp) => resp.json())
      .then((data)=>
        setUploadMessage(alertMessage(data)))
      .then((error:any) => console.log(error));
  };
  const addTags = (event: any) => {
    event.preventDefault();
    if (event.target.value !== '') {
      setTags([...tags, event.target.value]);
      event.target.value = '';
    }
  };
  const removeTags = (indexToRemove: any) => {
    setTags([...tags.filter((val, index) =>
      index !== indexToRemove)]);
  };
  const typeset=(e:any) =>{
    setType(e.target.text);
  };

  const [isOpen, set] = useState(false);
  const ontoggle =
  (isOpen: React.SetStateAction<boolean>) => set(isOpen);
  const onSelect = () => set(!isOpen);
  const dropdownItems = [
    <DropdownItem key="link"
      onClick={typeset}
    >
      Task</DropdownItem>,
    <DropdownItem key="action"
      onClick={typeset}
    >
          Pipeline
    </DropdownItem>,
  ];


  return (
    <Form id = "form" className="flex-size" onSubmit={submitdata}
      style = {{marginLeft: '5em'}}>
      <h1 style={{fontSize: '2em',
        fontFamily: 'bold'}}>Import Resources</h1>
      <FormGroup
        label="Name"
        isRequired
        fieldId="task-name"
        helperText="Please provide metadata name
         of  Task/Pipeline YAML file."
      >
        <TextInput
          isRequired
          type="text"
          id="task-name"
          name="task-name"
          autoComplete="off"
        />
      </FormGroup>
      <FormGroup
        isRequired label="Description"
        helperText="Please fill the description
        of your Task/pipeline."
        fieldId="description"
      >
        <TextArea style = {{height: '7em'}}
          name="description"
          id="description"
        />
      </FormGroup>
      <FormGroup label="Tags"
        isRequired fieldId="task-tag"
        helperText="Please provide
         tags name of your Task/Pipeline"
      >
        <div className="tags-input">
          <ChipGroup>
            {tags.map((chip, index) => (
              <Chip key={index}
                onClick={() => removeTags(index)} >
                {chip}
              </Chip>
            ))}
          </ChipGroup>
          <TextInput
            style={{marginTop: '0.3em'}}
            isRequired
            type="text"
            id="task-tags"
            name="task-tags"
            onKeyPress=
              {(event) =>
                (event.key === 'Enter' ? addTags(event) : null)}
            placeholder="Press enter to add tags"
            autoComplete="off"
          />
        </div>
      </FormGroup>
      <FormGroup label="Type"
        isRequired
        fieldId="task-tag">
        <div>
          <Dropdown style = {{backgroundColor: 'whitesmoke'}}
            onSelect = {onSelect}
            toggle={<DropdownToggle onToggle={ontoggle}>
              {type}</DropdownToggle>} // provide task type by default
            isOpen = {isOpen}
            dropdownItems={dropdownItems}
          />
        </div>
      </FormGroup>
      <FormGroup
        label="Github"
        isRequired
        helperText="Please provide the
         github link of your Task/Pipeline"
        fieldId="tasklink"
      >
        <TextInput
          name="tasklink"
          id="tasklink"
        />
      </FormGroup>
      <b> {uploadMessage} </b>
      {load }
      <ActionGroup>
        <Button id="Button"
          variant="primary"
          type="submit"
        >Submit</Button>
        <Link to="/" >
          <Button variant="secondary"
            type="submit">Cancel</Button>
        </Link>
      </ActionGroup>
    </Form>
  );
};
export default UploadTask;

