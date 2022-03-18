package v1

#ImagesData: {
	containers: [string]: {
		image: string
		sidecars: [string]: {
			image: string
		}
	}
	images: [string]: {
		image: string
	}
}

#BuilderSpec: {
	containers: [string]: {
		image?: string
		build?: #BuildSpec
		sidecars: [string]: {
			image?: string
			build?: #BuildSpec
		}
	}
	images: [string]: {
		image?: string
		build?: #BuildSpec
	}
}
