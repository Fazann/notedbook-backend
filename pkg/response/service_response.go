package response

import (
	"math"
	"net/http"
)

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

// SvcRespPage builds a paginated 200 OK response, computing TotalPage and the IsNext / IsPrev
// flags from page, limit and total so callers don't repeat the math.
func SvcRespPage(data any, page, limit int, total int64) SvcResWithPaginate {
	totalPage := int(math.Ceil(float64(total) / float64(limit)))
	return SvcResWithPaginate{
		Data:       data,
		StatusCode: http.StatusOK,
		IsNext:     page < totalPage,
		IsPrev:     page > 1,
		Total:      total,
		TotalPage:  totalPage,
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
