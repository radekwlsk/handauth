package samples

type UserSample struct {
	user   string
	sample *Sample
}

func (s *UserSample) Sample() *Sample {
	return s.sample
}

func NewUserSample(username, filename string) (*UserSample, error) {
	im, err := NewSample(filename)
	if err != nil {
		return nil, err
	}
	s := &UserSample{
		user:   username,
		sample: im,
	}
	return s, err
}

func (s *UserSample) Copy() *UserSample {
	return &UserSample{
		user:   s.user,
		sample: s.sample.Copy(),
	}
}

func (s *UserSample) Preprocess() {
	s.sample.Preprocess(0.0)
}

func (s *UserSample) Save(dir, filename string, show bool) {
	s.sample.Save(dir, filename, show)
}

func (s *UserSample) Close() {
	s.sample.Close()
}
