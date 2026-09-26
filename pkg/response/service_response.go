package response

type SvcRes struct {
	Data       interface{}
	Error      error
	StatusCode int
}

type SvcResWithPaginate struct {
	Data       interface{}
	Error      error
	StatusCode int
	IsNext     bool
	IsPrev     bool
	Total      int64
	TotalPage  int
}

func SvcRespErr(err error, statusCode int) SvcRes {
	return SvcRes{
		Error:      err,
		StatusCode: statusCode,
	}
}

func SvcRespSuccess(data any, statusCode int) SvcRes {
	return SvcRes{
		Data:       data,
		StatusCode: statusCode,
	}
}

func SvcResSucWithPage(res Paginate) SvcResWithPaginate {
	return SvcResWithPaginate{
		Data:       res.Data,
		Error:      nil,
		StatusCode: res.StatusCode,
		IsNext:     res.IsNext,
		IsPrev:     res.IsPrev,
		Total:      res.Total,
		TotalPage:  res.TotalPage,
	}
}
