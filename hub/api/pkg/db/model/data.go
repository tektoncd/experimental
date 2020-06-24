package model

//func initData(db *gorm.DB) {

//cat := &Catalog{
//Name:     "Tekton",
//Type:     "official",
//Owner:    "tektoncd",
//URL:      "https://github.com/Pipelines-Marketplace/catalog",
//Revision: "master",
//}
//db.Create(cat)

////for _, resource := range initResources {
////db.Model(&cat).Association("Resources").Append(&resource)
////}

////for _, resourceTag := range initResourceTags {
////db.Create(&resourceTag)
////}
//}

// var initResources = []Resource{
//
// 	Resource{
// 		Name:   "Buildah",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "Buildah task builds source into a container image and then pushes it to a container registry.\n" +
// 					" Buildah Task builds source into a container image using Project Atomic's Buildah build tool." +
// 					" It uses Buildah's support for building from Dockerfiles, using its buildah bud command." +
// 					" This command executes the directives in the Dockerfile to assemble a container image," +
// 					" then pushes that image to a container registry.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/buildah/0.1/buildah.yaml",
// 			},
// 			{
// 				Description: "Buildah task builds source into a container image and then pushes it to a container registry.\n" +
// 					" Buildah Task builds source into a container image using Project Atomic's Buildah build tool." +
// 					" It uses Buildah's support for building from Dockerfiles, using its buildah bud command." +
// 					" This command executes the directives in the Dockerfile to assemble a container image," +
// 					" then pushes that image to a container registry.",
// 				Version: "0.2",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/buildah/0.2/buildah.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "gcs-create-bucket",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "A Task that creates a new GCS bucket.\n" +
// 					" These Tasks are for copying to and from GCS buckets from Pipelines." +
// 					" These Tasks do a similar job to the GCS PipelineResource and are intended as its replacement.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/gcs-create-bucket/0.1/gcs-create-bucket.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "gcs-delete-bucket",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "A Task that deletes a GCS bucket.\n" +
// 					" These Tasks are for copying to and from GCS buckets from Pipelines." +
// 					" These Tasks do a similar job to the GCS PipelineResource and are intended as its replacement.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/gcs-delete-bucket/0.1/gcs-delete-bucket.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "gcs-download",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "A Task that fetches files or directories from a GCS bucket and puts them on a Workspace.\n" +
// 					" These Tasks are for copying to and from GCS buckets from Pipelines." +
// 					" These Tasks do a similar job to the GCS PipelineResource and are intended as its replacement.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/gcs-download/0.1/gcs-download.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "gcs-upload",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "A Task that uploads files or directories from a Workspace to a GCS bucket.\n" +
// 					" These Tasks are for copying to and from GCS buckets from Pipelines." +
// 					" These Tasks do a similar job to the GCS PipelineResource and are intended as its replacement.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/gcs-upload/0.1/gcs-upload.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "git-clone",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "Git-clone Tasks are Git tasks to work with repositories used by other tasks in your Pipeline.\n" +
// 					" The git-clone Task will clone a repo from the provided url into the" +
// 					" output Workspace. By default the repo will be cloned into a subdirectory" +
// 					" called \"src\" in your Workspace. You can clone into an alternative" +
// 					" subdirectory by setting this Task's subdirectory param.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/git-clone/0.1/git-clone.yaml",
// 			},
// 			{
// 				Description: "Git-clone Tasks are Git tasks to work with repositories used by other tasks in your Pipeline.\n" +
// 					" The git-clone Task will clone a repo from the provided url into the" +
// 					" output Workspace. By default the repo will be cloned into a subdirectory" +
// 					" called \"src\" in your Workspace. You can clone into an alternative" +
// 					" subdirectory by setting this Task's subdirectory param.",
// 				Version: "0.2",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/git-clone/0.2/git-clone.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "kaniko",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "This Task builds source into a container image using Google's kaniko tool.\n" +
// 					" Kaniko doesn't depend on a Docker daemon and executes each" +
// 					" command within a Dockerfile completely in userspace. This enables" +
// 					" building container images in environments that can't easily or" +
// 					" securely run a Docker daemon, such as a standard Kubernetes cluster.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/kaniko/0.1/kaniko.yaml",
// 			},
// 			{
// 				Description: "This Task builds source into a container image using Google's kaniko tool.\n" +
// 					" Kaniko doesn't depend on a Docker daemon and executes each" +
// 					" command within a Dockerfile completely in userspace. This enables" +
// 					" building container images in environments that can't easily or" +
// 					" securely run a Docker daemon, such as a standard Kubernetes cluster.",
// 				Version: "0.2",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/kaniko/0.2/kaniko.yaml",
// 			},
// 		},
// 	},
// 	Resource{
// 		Name:   "s2i",
// 		Type:   "task",
// 		Rating: 0,
// 		Versions: []ResourceVersion{
// 			{
// 				Description: "S2I Task builds source into a container image.\n" +
// 					" Source-to-Image (S2I) is a toolkit and workflow for building reproducible" +
// 					" container images from source code. S2I produces images by injecting" +
// 					" source code into a base S2I container image and letting the container" +
// 					" prepare that source code for execution. The base S2I container images contains" +
// 					" the language runtime and build tools needed for building and running the source code.",
// 				Version: "0.1",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/s2i/0.1/s2i.yaml",
// 			},
// 			{
// 				Description: "S2I Task builds source into a container image.\n" +
// 					" Source-to-Image (S2I) is a toolkit and workflow for building reproducible" +
// 					" container images from source code. S2I produces images by injecting" +
// 					" source code into a base S2I container image and letting the container" +
// 					" prepare that source code for execution. The base S2I container images contains" +
// 					" the language runtime and build tools needed for building and running the source code.",
// 				Version: "0.2",
// 				URL:     "https://github.com/Pipelines-Marketplace/catalog/blob/master/task/s2i/0.2/s2i.yaml",
// 			},
// 		},
// 	},
// }
//
// var initResourceTags = []ResourceTag{
// 	ResourceTag{
// 		ResourceID: 1,
// 		TagID:      1,
// 	},
// 	ResourceTag{
// 		ResourceID: 2,
// 		TagID:      9,
// 	},
// 	ResourceTag{
// 		ResourceID: 3,
// 		TagID:      9,
// 	},
// 	ResourceTag{
// 		ResourceID: 4,
// 		TagID:      9,
// 	},
// 	ResourceTag{
// 		ResourceID: 5,
// 		TagID:      9,
// 	},
// 	ResourceTag{
// 		ResourceID: 7,
// 		TagID:      1,
// 	},
// 	ResourceTag{
// 		ResourceID: 8,
// 		TagID:      1,
// 	},
// }
//
